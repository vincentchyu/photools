package gpxmatch

import (
	"reflect"
	"testing"
)

func TestExtractDateFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"hiking-中国-广东-白云山-20250322.gpx", "2025-03-22"},
		{"walking-海珠湖-2025-05-01.gpx", "2025-05-01"},
		{"hiking_2026_06_13.gpx", "2026-06-13"},
		{"hiking-route.gpx", ""},
		{"20250322.gpx", "2025-03-22"},
		{"invalid-20251340.gpx", ""}, // 非法月份日期
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := ExtractDateFromFilename(tt.filename)
			if got != tt.expected {
				t.Errorf("ExtractDateFromFilename(%q) = %q, expected %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestFilterGPXFilesByDate(t *testing.T) {
	allGPX := []string{
		"/gpx/hiking-白云山-20250321.gpx", // 前一天
		"/gpx/hiking-白云山-20250322.gpx", // 当天
		"/gpx/hiking-白云山-20250323.gpx", // 后一天
		"/gpx/hiking-九华山-20250705.gpx", // 遥远日期（应排除）
		"/gpx/hiking-新疆-20260613.gpx",  // 遥远日期（应排除）
		"/gpx/hiking-nodate.gpx",       // 无日期（应兜底保留）
	}

	// 1. 正常拍摄时间：2025-03-22 14:00:00 -> 应匹配 0321, 0322, 0323 及 nodate
	res1 := FilterGPXFilesByDate(allGPX, "2025:03:22 14:00:00")
	expected1 := []string{
		"/gpx/hiking-白云山-20250321.gpx",
		"/gpx/hiking-白云山-20250322.gpx",
		"/gpx/hiking-白云山-20250323.gpx",
		"/gpx/hiking-nodate.gpx",
	}
	if !reflect.DeepEqual(res1, expected1) {
		t.Errorf("FilterGPXFilesByDate failed:\n got:      %v\n expected: %v", res1, expected1)
	}

	// 2. 月级别降级用例：照片拍摄于 2025-10-02，但轨迹为合并的 2025-10-04（相差2天，日邻近未命中，但同在2025-10月）
	wutaiGPX := []string{
		"/gpx/hiking-中国-山西-五台山逆朝台-20251004.gpx", // 目标合并轨迹 (2025-10)
		"/gpx/hiking-白云山-20250322.gpx",          // 其他月份 (应排除)
		"/gpx/hiking-新疆-20260613.gpx",           // 其他年份 (应排除)
		"/gpx/hiking-nodate.gpx",                // 无日期 (兜底保留)
	}
	res2 := FilterGPXFilesByDate(wutaiGPX, "2025:10:02 09:30:00")
	expected2 := []string{
		"/gpx/hiking-中国-山西-五台山逆朝台-20251004.gpx",
		"/gpx/hiking-nodate.gpx",
	}
	if !reflect.DeepEqual(res2, expected2) {
		t.Errorf("FilterGPXFilesByDate month fallback failed:\n got:      %v\n expected: %v", res2, expected2)
	}

	// 3. 无效拍摄时间 -> 应兜底返回全量
	res3 := FilterGPXFilesByDate(allGPX, "")
	if !reflect.DeepEqual(res3, allGPX) {
		t.Errorf("FilterGPXFilesByDate with empty date should return all gpx")
	}

	// 4. 只有一个 GPX 时直接返回
	single := []string{"/gpx/track.gpx"}
	res4 := FilterGPXFilesByDate(single, "2025:03:22 14:00:00")
	if !reflect.DeepEqual(res4, single) {
		t.Errorf("FilterGPXFilesByDate with single file should return original")
	}
}
