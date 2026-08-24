package engine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vincentchyu/photo-processing/internal/domain"
)

// BackupAssetGroups 将待处理资产组完整复制快照到指定的备份目录中
func BackupAssetGroups(groups []domain.AssetGroup, sourceRoot, targetBackupDir string) (int, error) {
	if len(groups) == 0 {
		return 0, nil
	}

	if err := os.MkdirAll(targetBackupDir, 0o755); err != nil {
		return 0, fmt.Errorf("创建备份目录失败 (%s): %w", targetBackupDir, err)
	}

	copiedCount := 0
	for _, g := range groups {
		for _, srcPath := range g.AllFiles() {
			if srcPath == "" {
				continue
			}

			// 计算相对于 sourceRoot 的相对路径以保持目录层级
			rel, err := filepath.Rel(sourceRoot, srcPath)
			if err != nil || rel == "" || rel == "." {
				rel = filepath.Base(srcPath)
			}

			destPath := filepath.Join(targetBackupDir, rel)
			if err := copyFilePreserve(srcPath, destPath); err != nil {
				return copiedCount, fmt.Errorf("复制备份文件失败 [%s -> %s]: %w", srcPath, destPath, err)
			}
			copiedCount++
		}
	}

	return copiedCount, nil
}

// RestoreBackup 从备份目录还原文件到目标 Inbox 目录中
func RestoreBackup(backupDir, targetInboxDir string) (int, error) {
	if fi, err := os.Stat(backupDir); err != nil || !fi.IsDir() {
		return 0, fmt.Errorf("备份目录不存在或无法读取: %s", backupDir)
	}

	if err := os.MkdirAll(targetInboxDir, 0o755); err != nil {
		return 0, fmt.Errorf("创建目标目录失败 (%s): %w", targetInboxDir, err)
	}

	restoredCount := 0
	err := filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(backupDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(targetInboxDir, rel)
		if err := copyFilePreserve(path, destPath); err != nil {
			return fmt.Errorf("还原文件失败 [%s -> %s]: %w", path, destPath, err)
		}
		restoredCount++
		return nil
	})

	if err != nil {
		return restoredCount, err
	}

	return restoredCount, nil
}

func copyFilePreserve(src, dst string) error {
	srcFi, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcFi.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// 保持原文件的修改时间
	_ = os.Chtimes(dst, srcFi.ModTime(), srcFi.ModTime())
	return nil
}
