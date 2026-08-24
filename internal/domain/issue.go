package domain

// IssueKind 定义待处理项或失败分类
type IssueKind string

const (
	IssueKindMissingPair IssueKind = "missing_pair" // 缺少同名配对 JPG
	IssueKindTrackGap    IssueKind = "track_gap"    // 拍摄时间未在 GPX 轨迹范围内
	IssueKindFailure     IssueKind = "failure"      // 元数据解析或写入失败
	IssueKindMissingDate IssueKind = "missing_date" // 无法读取拍摄日期
	IssueKindConflict    IssueKind = "conflict"     // 归档目标目录已存在同名冲突文件
)

// Issue 记录单个资产组的异常详情与改进建议
type Issue struct {
	Kind               IssueKind  `json:"kind"`
	Reason             string     `json:"reason"`
	Suggestion         string     `json:"suggestion"`
	Asset              AssetGroup `json:"asset"`
	PhotoTime          string     `json:"photo_time,omitempty"`
	PhotoOffset        string     `json:"photo_offset,omitempty"`
	ReferencedGPXFiles []string   `json:"referenced_gpx_files,omitempty"`
	FailedStage        string     `json:"failed_stage,omitempty"`   // 发生故障的初始阶段
	BlockedStages      []string   `json:"blocked_stages,omitempty"` // 因而被安全熔断跳过的后续阶段列表
	CurrentStatus      string     `json:"current_status,omitempty"` // 资产物理文件当前存放状态
}

// TaskSummary 汇总任务运行结算指标
type TaskSummary struct {
	TotalAssets int `json:"total_assets"`
	Success     int `json:"success"`
	Pending     int `json:"pending"`
	Failed      int `json:"failed"`
	Skipped     int `json:"skipped"`
}
