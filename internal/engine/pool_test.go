package engine

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_ExecuteOrdered(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	var processedCount atomic.Int32

	pool := NewWorkerPool[int, string](4)
	results := pool.Execute(context.Background(), items, func(ctx context.Context, item int) string {
		processedCount.Add(1)
		time.Sleep(10 * time.Millisecond)
		return fmt.Sprintf("result-%d", item)
	})

	if len(results) != len(items) {
		t.Fatalf("期望得到 %d 个结果，实际得到 %d", len(items), len(results))
	}

	if int(processedCount.Load()) != len(items) {
		t.Errorf("期望处理 %d 项，实际处理 %d", len(items), processedCount.Load())
	}

	for i, res := range results {
		expected := fmt.Sprintf("result-%d", items[i])
		if res != expected {
			t.Errorf("索引 %d 结果不匹配: 期望 %q, 实际 %q", i, expected, res)
		}
	}
}
