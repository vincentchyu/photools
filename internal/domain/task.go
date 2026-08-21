package domain

import "context"

// PlanItem 预检计划中的单项
type PlanItem struct {
	Asset         AssetGroup `json:"asset"`
	Action        string     `json:"action"`         // 将要执行的动作描述
	TargetPath    string     `json:"target_path"`    // 预期归档目录或文件
	WillWriteGPS  bool       `json:"will_write_gps"` // 是否会写入 GPS
	EstimatedTime string     `json:"estimated_time"` // 提取到的拍摄时间
	Warning       string     `json:"warning"`        // 预警信息（如有）
}

// PlanResult 包含预检 (Dry-Run) 的完整结果
type PlanResult struct {
	Items         []PlanItem `json:"items"`
	TotalAssets   int        `json:"total_assets"`
	ReadyCount    int        `json:"ready_count"`
	PendingCount  int        `json:"pending_count"`
	WarningsCount int        `json:"warnings_count"`
	Details       string     `json:"details"`
}

// Task 定义通用的摄影处理任务接口
type Task interface {
	// Name 返回任务中文名称
	Name() string
	// Description 返回任务详细描述
	Description() string
	// Stages 返回该任务专有的流水线阶段列表
	Stages() []PipelineStage
	// Plan 执行预检（Dry-Run），不产生实际写入与移动
	Plan(ctx context.Context) (*PlanResult, error)
	// Execute 执行真实流水线，实时通过 eventCh 发送进度事件
	Execute(ctx context.Context, eventCh chan<- ProgressEvent) (*TaskSummary, []Issue, error)
}
