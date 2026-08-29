package domain

import "github.com/vincentchyu/photools/common"

// IssueKind 定义待处理项或失败分类 (指向 common.IssueKind)
type IssueKind = common.IssueKind

const (
	IssueKindMissingPair = common.IssueKindMissingPair
	IssueKindTrackGap    = common.IssueKindTrackGap
	IssueKindFailure     = common.IssueKindFailure
	IssueKindMissingDate = common.IssueKindMissingDate
	IssueKindConflict    = common.IssueKindConflict
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
