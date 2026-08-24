package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vincentchyu/photo-processing/internal/domain"
)

func TestBackupAndRestoreAssetGroups(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "photools_backup_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceInbox := filepath.Join(tempDir, "Inbox")
	subFolder := filepath.Join(sourceInbox, "0517")
	_ = os.MkdirAll(subFolder, 0o755)

	rawFile := filepath.Join(subFolder, "DSC_001.NEF")
	jpgFile := filepath.Join(subFolder, "DSC_001.JPG")
	xmpFile := filepath.Join(subFolder, "DSC_001.xmp")

	_ = os.WriteFile(rawFile, []byte("fake raw content"), 0o644)
	_ = os.WriteFile(jpgFile, []byte("fake jpg content"), 0o644)
	_ = os.WriteFile(xmpFile, []byte("fake xmp content"), 0o644)

	groups := []domain.AssetGroup{
		{
			BaseName: "DSC_001",
			Dir:      subFolder,
			RawPath:  rawFile,
			JPGPath:  jpgFile,
			XMPPath:  xmpFile,
		},
	}

	backupDir := filepath.Join(tempDir, "Inbox_bak")

	// 1. 测试备份
	count, err := BackupAssetGroups(groups, sourceInbox, backupDir)
	if err != nil {
		t.Fatalf("BackupAssetGroups 失败: %v", err)
	}
	if count != 3 {
		t.Errorf("期望备份 3 个文件, 实际: %d", count)
	}

	// 验证备份文件是否存在
	bakRaw := filepath.Join(backupDir, "0517", "DSC_001.NEF")
	bakJPG := filepath.Join(backupDir, "0517", "DSC_001.JPG")
	bakXMP := filepath.Join(backupDir, "0517", "DSC_001.xmp")

	if _, err := os.Stat(bakRaw); err != nil {
		t.Errorf("备份的 RAW 文件不存在: %v", err)
	}
	if _, err := os.Stat(bakJPG); err != nil {
		t.Errorf("备份的 JPG 文件不存在: %v", err)
	}
	if _, err := os.Stat(bakXMP); err != nil {
		t.Errorf("备份的 XMP 文件不存在: %v", err)
	}

	// 2. 模拟清空原 Inbox
	_ = os.RemoveAll(sourceInbox)

	// 3. 测试还原
	restoredCount, err := RestoreBackup(backupDir, sourceInbox)
	if err != nil {
		t.Fatalf("RestoreBackup 失败: %v", err)
	}
	if restoredCount != 3 {
		t.Errorf("期望还原 3 个文件, 实际: %d", restoredCount)
	}

	if _, err := os.Stat(rawFile); err != nil {
		t.Errorf("还原后的 RAW 文件不存在: %v", err)
	}
	if _, err := os.Stat(jpgFile); err != nil {
		t.Errorf("还原后的 JPG 文件不存在: %v", err)
	}
	if _, err := os.Stat(xmpFile); err != nil {
		t.Errorf("还原后的 XMP 文件不存在: %v", err)
	}
}
