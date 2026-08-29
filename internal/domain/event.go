package domain

import (
	"time"

	"github.com/vincentchyu/photools/common"
)

// EventLevel 定义事件严重级别 (指向 common.EventLevel)
type EventLevel = common.EventLevel

const (
	LevelInfo    = common.LevelInfo
	LevelSuccess = common.LevelSuccess
	LevelWarn    = common.LevelWarn
	LevelError   = common.LevelError
)

// PipelineStage 定义当前流水线所处阶段 (指向 common.PipelineStage)
type PipelineStage = common.PipelineStage

const (
	StageInit        = common.StageInit
	StageDiscover    = common.StageDiscover
	StagePrecheck    = common.StagePrecheck
	StageGeotag      = common.StageGeotag
	StageInterpolate = common.StageInterpolate
	StageGeocode     = common.StageGeocode
	StageSync        = common.StageSync
	StageArchive     = common.StageArchive
	StageBackup      = common.StageBackup
	StageRestore     = common.StageRestore
	StageSummary     = common.StageSummary
	StageComplete    = common.StageComplete
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
