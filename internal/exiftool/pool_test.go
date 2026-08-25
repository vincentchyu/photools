package exiftool

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStayOpenPoolBasic(t *testing.T) {
	pool, err := NewStayOpenPool(
		StayOpenConfig{
			MaxWorkers:     2,
			CommandTimeout: 5 * time.Second,
		},
	)
	if err != nil {
		t.Fatalf("创建 StayOpenPool 失败: %v", err)
	}
	defer func() {
		_ = pool.Close()
	}()

	// 1. 执行 -ver 测试
	output, err := pool.Execute("-ver")
	if err != nil {
		t.Fatalf("pool.Execute -ver 失败: %v", err)
	}
	verStr := strings.TrimSpace(string(output))
	if len(verStr) == 0 {
		t.Errorf("期望返回 ExifTool 版本号，实际为空")
	}

	// 2. 连续多次执行验证 Worker 复用
	for i := range 5 {
		out, err := pool.Execute("-ver")
		if err != nil {
			t.Fatalf("第 %d 次执行失败: %v", i+1, err)
		}
		if strings.TrimSpace(string(out)) != verStr {
			t.Errorf("第 %d 次执行版本号不一致: %s vs %s", i+1, string(out), verStr)
		}
	}
}

func TestStayOpenPoolConcurrent(t *testing.T) {
	pool, err := NewStayOpenPool(
		StayOpenConfig{
			MaxWorkers:     4,
			CommandTimeout: 10 * time.Second,
		},
	)
	if err != nil {
		t.Fatalf("创建 StayOpenPool 失败: %v", err)
	}
	defer func() {
		_ = pool.Close()
	}()

	var wg sync.WaitGroup
	workers := 10
	iterations := 5

	for w := range workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := range iterations {
				out, err := pool.Execute("-ver")
				if err != nil {
					t.Errorf("Worker %d 在第 %d 次请求出错: %v", workerID, i, err)
					return
				}
				if len(strings.TrimSpace(string(out))) == 0 {
					t.Errorf("Worker %d 在第 %d 次请求返回空输出", workerID, i)
					return
				}
			}
		}(w)
	}

	wg.Wait()
}

func TestStayOpenPoolAutoRecovery(t *testing.T) {
	pool, err := NewStayOpenPool(
		StayOpenConfig{
			MaxWorkers:     1,
			CommandTimeout: 5 * time.Second,
		},
	)
	if err != nil {
		t.Fatalf("创建 StayOpenPool 失败: %v", err)
	}
	defer func() {
		_ = pool.Close()
	}()

	// 第一次正常执行
	out1, err := pool.Execute("-ver")
	if err != nil {
		t.Fatalf("首次执行失败: %v", err)
	}

	// 人为借出 worker 并强行 kill 其底层进程以模拟崩溃
	w, err := pool.acquireWorker()
	if err != nil {
		t.Fatalf("acquireWorker 失败: %v", err)
	}
	_ = w.cmd.Process.Kill()
	w.broken = true
	pool.releaseWorker(w)

	// 再次执行，验证进程池是否能够自愈重建新进程并成功响应
	out2, err := pool.Execute("-ver")
	if err != nil {
		t.Fatalf("崩溃后自愈执行失败: %v", err)
	}
	if strings.TrimSpace(string(out1)) != strings.TrimSpace(string(out2)) {
		t.Errorf("自愈后输出不匹配: %s vs %s", string(out1), string(out2))
	}
}

func TestPoolRunnerCombinedOutput(t *testing.T) {
	runner, err := NewPoolRunner(
		StayOpenConfig{
			MaxWorkers: 2,
		},
	)
	if err != nil {
		t.Fatalf("创建 PoolRunner 失败: %v", err)
	}
	defer func() {
		_ = runner.Close()
	}()

	// 1. 测试 exiftool 命令（走常驻池）
	out, err := runner.CombinedOutput("exiftool", "-ver")
	if err != nil {
		t.Fatalf("runner CombinedOutput exiftool 失败: %v", err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Errorf("期望返回版本号")
	}

	// 2. 测试非 exiftool 命令（走 exec.Command 降级）
	echoOut, err := runner.CombinedOutput("echo", "hello-world")
	if err != nil {
		t.Fatalf("runner CombinedOutput echo 失败: %v", err)
	}
	if !strings.Contains(string(echoOut), "hello-world") {
		t.Errorf("echo 输出异常: %s", string(echoOut))
	}
}

func TestStayOpenPoolClose(t *testing.T) {
	pool, err := NewStayOpenPool(
		StayOpenConfig{
			MaxWorkers: 2,
		},
	)
	if err != nil {
		t.Fatalf("创建 StayOpenPool 失败: %v", err)
	}

	// 正常关闭
	if err := pool.Close(); err != nil {
		t.Errorf("关闭 pool 失败: %v", err)
	}

	// 重复关闭应幂等安全
	if err := pool.Close(); err != nil {
		t.Errorf("重复关闭 pool 报错: %v", err)
	}

	// 关闭后再执行应返回错误
	_, err = pool.Execute("-ver")
	if err == nil {
		t.Errorf("关闭后执行应报错")
	}
}

func TestDefaultRunner(t *testing.T) {
	CloseDefaultPool()
	defer CloseDefaultPool()

	r := DefaultRunner()
	if r == nil {
		t.Fatal("DefaultRunner 返回 nil")
	}

	out, err := r.CombinedOutput("exiftool", "-ver")
	if err != nil {
		t.Fatalf("DefaultRunner 运行失败: %v", err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Errorf("DefaultRunner 期望返回版本号")
	}
}
