//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/vincentchyu/photo-processing/internal/geodata"
)

func main() {
	fmt.Println("🚀 正在从 GeoNames 官方开放源构建离线大洲地理数据包...")

	mgr, err := geodata.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	err = mgr.InstallAll(ctx, func(msg string) {
		fmt.Println(msg)
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "构建出错: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✨ 构建完成！数据已输出至: %s\n", mgr.GetDataDir())
}
