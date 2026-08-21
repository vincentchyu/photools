package domain

import "time"

// EventLevel 定义事件严重级别
type EventLevel string

const (
	LevelInfo    EventLevel = "info"
	LevelSuccess EventLevel = "success"
	LevelWarn    EventLevel = "warn"
	LevelError   EventLevel = "error"
)

// PipelineStage 定义当前流水线所处阶段
type PipelineStage string

const (
	StageDiscover PipelineStage = "扫描资产"
	StagePrecheck PipelineStage = "预检校验"
	StageGeotag   PipelineStage = "写入GPS"
	StageSync     PipelineStage = "同步附属文件"
	StageArchive  PipelineStage = "归档重命名"
	StageComplete PipelineStage = "任务完成"
)

// ProgressEvent 表示在任务执行期间发送给 UI/TUI/日志的结构化事件
type ProgressEvent struct {
	Timestamp time.Time     `json:"timestamp"`
	Stage     PipelineStage `json:"stage"`
	Level     EventLevel    `json:"level"`
	Message   string        `json:"message"`

	// 当前进度指示 (如 5/20)
	CurrentIndex int `json:"current_index,omitempty"`
	TotalItems   int `json:"total_items,omitempty"`

	// 关联的资产
	Asset *AssetGroup `json:"asset,omitempty"`
	Issue *Issue      `json:"issue,omitempty"`
}

// NewInfoEvent 构造一个普通信息事件
func NewInfoEvent(stage PipelineStage, msg string) ProgressEvent {
	return ProgressEvent{
		Timestamp: time.Now(),
		Stage:     stage,
		Level:     LevelInfo,
		Message:   msg,
	}
}

// NewAssetProgressEvent 构造资产维度的进度事件
func NewAssetProgressEvent(stage PipelineStage, level EventLevel, msg string, asset *AssetGroup, current, total int) ProgressEvent {
	return ProgressEvent{
		Timestamp:    time.Now(),
		Stage:        stage,
		Level:        level,
		Message:      msg,
		CurrentIndex: current,
		TotalItems:   total,
		Asset:        asset,
	}
}
