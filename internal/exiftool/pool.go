package exiftool

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// StayOpenConfig 常驻进程池配置
type StayOpenConfig struct {
	// ExifToolPath 可执行文件路径，为空时自动探测
	ExifToolPath string
	// MaxWorkers 最大 Worker 进程数，默认根据 CPU 核心数自适应 (2 ~ 16)
	MaxWorkers int
	// CommandTimeout 单次指令超时时间，默认 30s
	CommandTimeout time.Duration
}

// StayOpenWorker 单个 ExifTool 常驻子进程
type StayOpenWorker struct {
	id       int
	binPath  string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	stderr   *bytes.Buffer
	stderrMu sync.Mutex
	broken   bool
	closed   bool
	mu       sync.Mutex
}

// newStayOpenWorker 启动一个新的常驻 ExifTool 进程
func newStayOpenWorker(id int, binPath string) (*StayOpenWorker, error) {
	if binPath == "" {
		binPath = LocateExifTool()
	}

	cmd := exec.Command(binPath, "-stay_open", "True", "-@", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 exiftool stdin 管道失败: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("创建 exiftool stdout 管道失败: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdoutPipe.Close()
		return nil, fmt.Errorf("创建 exiftool stderr 管道失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdoutPipe.Close()
		_ = stderrPipe.Close()
		return nil, fmt.Errorf("启动常驻 exiftool 进程失败: %w", err)
	}

	w := &StayOpenWorker{
		id:      id,
		binPath: binPath,
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdoutPipe),
		stderr:  &bytes.Buffer{},
	}

	// 异步收集 stderr 输出
	go func() {
		buf := make([]byte, 4096)
		for {
			n, rErr := stderrPipe.Read(buf)
			if n > 0 {
				w.stderrMu.Lock()
				w.stderr.Write(buf[:n])
				w.stderrMu.Unlock()
			}
			if rErr != nil {
				break
			}
		}
	}()

	return w, nil
}

// Execute 向常驻 Worker 写入参数并同步等待其输出结果直至匹配就绪标记
func (w *StayOpenWorker) Execute(reqID uint64, args []string, timeout time.Duration) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.broken || w.closed {
		return nil, errors.New("exiftool worker 进程已失效或已关闭")
	}

	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// 重置并清空 stderr 缓存
	w.stderrMu.Lock()
	w.stderr.Reset()
	w.stderrMu.Unlock()

	// 构造 stdin 数据流（参数逐行写入，以 -execute<reqID>\n 结尾）
	var payload bytes.Buffer
	for _, arg := range args {
		// 去除参数内部可能的换行以防破坏协议
		cleanArg := strings.ReplaceAll(arg, "\r", "")
		cleanArg = strings.ReplaceAll(cleanArg, "\n", " ")
		payload.WriteString(cleanArg)
		payload.WriteByte('\n')
	}
	readyTag := fmt.Sprintf("{ready%d}", reqID)
	payload.WriteString(fmt.Sprintf("-execute%d\n", reqID))

	// 写入 stdin
	if _, err := w.stdin.Write(payload.Bytes()); err != nil {
		w.broken = true
		return nil, fmt.Errorf("向 exiftool 写入指令失败: %w", err)
	}

	// 设立结果通道与超时控制
	type execResult struct {
		output []byte
		err    error
	}
	resChan := make(chan execResult, 1)

	go func() {
		var outBuf bytes.Buffer
		for {
			line, rErr := w.stdout.ReadBytes('\n')
			if len(line) > 0 {
				trimmedLine := strings.TrimRight(string(line), "\r\n")
				if trimmedLine == readyTag || strings.HasPrefix(trimmedLine, readyTag) {
					// 到达当前请求的结束标记
					resChan <- execResult{output: outBuf.Bytes(), err: nil}
					return
				}
				outBuf.Write(line)
			}
			if rErr != nil {
				resChan <- execResult{
					output: outBuf.Bytes(),
					err:    fmt.Errorf("读取 exiftool 输出异常 (进程可能已退出): %w", rErr),
				}
				return
			}
		}
	}()

	select {
	case res := <-resChan:
		if res.err != nil {
			w.broken = true
			return nil, res.err
		}

		w.stderrMu.Lock()
		stderrBytes := w.stderr.Bytes()
		var combined bytes.Buffer
		if len(stderrBytes) > 0 {
			combined.Write(stderrBytes)
		}
		combined.Write(res.output)
		w.stderrMu.Unlock()

		return combined.Bytes(), nil

	case <-time.After(timeout):
		w.broken = true
		_ = w.cmd.Process.Kill()
		return nil, fmt.Errorf("执行 exiftool 超时 (%v)", timeout)
	}
}

// Close 优雅退出常驻 ExifTool 进程
func (w *StayOpenWorker) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	if w.stdin != nil {
		_, _ = w.stdin.Write([]byte("-stay_open\nFalse\n"))
		_ = w.stdin.Close()
	}

	done := make(chan error, 1)
	go func() {
		done <- w.cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		if w.cmd.Process != nil {
			_ = w.cmd.Process.Kill()
		}
		return nil
	}
}

// StayOpenPool 管理一组常驻 ExifTool 进程的并发池
type StayOpenPool struct {
	config     StayOpenConfig
	workers    chan *StayOpenWorker
	allWorkers map[*StayOpenWorker]struct{}
	workerSeq  int
	reqCounter atomic.Uint64
	mu         sync.Mutex
	closed     bool
}

// NewStayOpenPool 创建一个新的 ExifTool 常驻进程池
func NewStayOpenPool(cfg StayOpenConfig) (*StayOpenPool, error) {
	if cfg.ExifToolPath == "" {
		cfg.ExifToolPath = LocateExifTool()
	}
	if cfg.MaxWorkers <= 0 {
		cpus := runtime.NumCPU()
		if cpus < 2 {
			cpus = 2
		} else if cpus > 16 {
			cpus = 16
		}
		cfg.MaxWorkers = cpus
	}
	if cfg.CommandTimeout <= 0 {
		cfg.CommandTimeout = 30 * time.Second
	}

	p := &StayOpenPool{
		config:     cfg,
		workers:    make(chan *StayOpenWorker, cfg.MaxWorkers),
		allWorkers: make(map[*StayOpenWorker]struct{}),
	}

	return p, nil
}

// acquireWorker 从池中借出一个健康的 Worker，支持按需创建与自愈
func (p *StayOpenPool) acquireWorker() (*StayOpenWorker, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, errors.New("exiftool 进程池已关闭")
	}
	p.mu.Unlock()

	// 优先尝试从通道获取空闲 worker
	select {
	case w := <-p.workers:
		if !w.broken && !w.closed {
			return w, nil
		}
		p.mu.Lock()
		delete(p.allWorkers, w)
		p.mu.Unlock()
		_ = w.Close()
	default:
	}

	// 若未获取到且当前池未满，则创建新 worker
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, errors.New("exiftool 进程池已关闭")
	}

	p.workerSeq++
	w, err := newStayOpenWorker(p.workerSeq, p.config.ExifToolPath)
	if err != nil {
		return nil, err
	}
	p.allWorkers[w] = struct{}{}
	return w, nil
}

// releaseWorker 将 Worker 归还给池，已损坏的 Worker 自动关闭
func (p *StayOpenPool) releaseWorker(w *StayOpenWorker) {
	if w == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed || w.broken || w.closed {
		delete(p.allWorkers, w)
		_ = w.Close()
		return
	}

	select {
	case p.workers <- w:
	default:
		// 超过池容量时优雅关闭多余的 worker 并从注册表中注销
		delete(p.allWorkers, w)
		_ = w.Close()
	}
}

// Execute 执行一次 ExifTool 指令
func (p *StayOpenPool) Execute(args ...string) ([]byte, error) {
	w, err := p.acquireWorker()
	if err != nil {
		return nil, err
	}
	defer p.releaseWorker(w)

	reqID := p.reqCounter.Add(1)
	return w.Execute(reqID, args, p.config.CommandTimeout)
}

// Close 关闭常驻进程池中所有子进程
func (p *StayOpenPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true

	// 收集当前登记的所有 worker (包含已借出与在队列中的)
	targets := make([]*StayOpenWorker, 0, len(p.allWorkers))
	for w := range p.allWorkers {
		targets = append(targets, w)
	}
	p.allWorkers = make(map[*StayOpenWorker]struct{})
	p.mu.Unlock()

	// 清空 channel
	close(p.workers)
	for range p.workers {
	}

	// 并发优雅关闭所有 worker
	var wg sync.WaitGroup
	for _, w := range targets {
		wg.Add(1)
		go func(worker *StayOpenWorker) {
			defer wg.Done()
			_ = worker.Close()
		}(w)
	}
	wg.Wait()
	return nil
}

// PoolRunner 实现 CommandRunner 接口的常驻进程池适配器
type PoolRunner struct {
	pool *StayOpenPool
}

// NewPoolRunner 创建一个基于常驻池的 CommandRunner
func NewPoolRunner(cfg StayOpenConfig) (*PoolRunner, error) {
	pool, err := NewStayOpenPool(cfg)
	if err != nil {
		return nil, err
	}
	return &PoolRunner{pool: pool}, nil
}

// CombinedOutput 执行命令，若是 exiftool 命令则通过常驻池流式管道执行
func (pr *PoolRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	if pr == nil || pr.pool == nil {
		return exec.Command(name, args...).CombinedOutput()
	}

	target := LocateExifTool()
	if name == "exiftool" || name == target || filepath.Base(name) == "exiftool" {
		return pr.pool.Execute(args...)
	}
	return exec.Command(name, args...).CombinedOutput()
}

// Close 关闭底层常驻进程池
func (pr *PoolRunner) Close() error {
	if pr != nil && pr.pool != nil {
		return pr.pool.Close()
	}
	return nil
}

var (
	defaultPoolRunner *PoolRunner
	defaultPoolOnce   sync.Once
	defaultPoolMu     sync.Mutex
)

// DefaultRunner 返回全局单例的 ExifTool 常驻进程池 Runner
func DefaultRunner() CommandRunner {
	defaultPoolMu.Lock()
	defer defaultPoolMu.Unlock()

	if defaultPoolRunner != nil {
		return defaultPoolRunner
	}

	runner, err := NewPoolRunner(StayOpenConfig{})
	if err == nil {
		defaultPoolRunner = runner
		return defaultPoolRunner
	}
	return ExecRunner{}
}

// CloseDefaultPool 优雅关闭全局默认常驻进程池
func CloseDefaultPool() {
	defaultPoolMu.Lock()
	defer defaultPoolMu.Unlock()
	if defaultPoolRunner != nil {
		_ = defaultPoolRunner.Close()
		defaultPoolRunner = nil
		defaultPoolOnce = sync.Once{}
	}
}
