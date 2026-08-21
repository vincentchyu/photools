package engine

import (
	"context"
	"runtime"
	"sync"
)

// WorkerPool 泛型保序并发调度器
type WorkerPool[T any, R any] struct {
	concurrency int
}

// NewWorkerPool 创建指定并发数的 WorkerPool
func NewWorkerPool[T any, R any](workers int) *WorkerPool[T, R] {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &WorkerPool[T, R]{concurrency: workers}
}

type poolJob[T any] struct {
	index int
	data  T
}

type poolResult[R any] struct {
	index int
	data  R
}

// Execute 并发处理数据列表，并按原始顺序返回所有结果
func (p *WorkerPool[T, R]) Execute(
	ctx context.Context,
	items []T,
	workerFn func(ctx context.Context, item T) R,
) []R {
	if len(items) == 0 {
		return nil
	}

	actualWorkers := p.concurrency
	if actualWorkers > len(items) {
		actualWorkers = len(items)
	}

	jobs := make(chan poolJob[T])
	results := make(chan poolResult[R], len(items))
	var wg sync.WaitGroup

	for i := 0; i < actualWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					res := workerFn(ctx, job.data)
					results <- poolResult[R]{
						index: job.index,
						data:  res,
					}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i, item := range items {
			select {
			case <-ctx.Done():
				return
			case jobs <- poolJob[T]{index: i, data: item}:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	ordered := make([]R, len(items))
	for res := range results {
		ordered[res.index] = res.data
	}

	return ordered
}
