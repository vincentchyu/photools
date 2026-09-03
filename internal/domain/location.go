package domain

import (
	"slices"
	"strings"

	"github.com/vincentchyu/photools/pkg/geocoding"
)

// LocationInfo 逆地理编码返回的地理位置详细信息 (映射到 pkg/geocoding.LocationInfo)
type LocationInfo = geocoding.LocationInfo

var (
	ToAlpha3 = geocoding.ToAlpha3
	ToAlpha2 = geocoding.ToAlpha2
)

// ExistingTags 存储媒体或 XMP 中读取到的现有标签，用于增量清洗与去重
type ExistingTags struct {
	HierarchicalSubject []string
	Subject             []string
	Keywords            []string
}

// CleanedLocationTags 封装清洗并合并后的最终唯一标签结果
type CleanedLocationTags struct {
	HierarchicalSubject []string
	Subject             []string
	Keywords            []string
}

// CleanAndMergeLocationTags 内存级地理标签清洗与用户自定义标签保护合并算法（纯函数、无副作用、绝对幂等）
func CleanAndMergeLocationTags(existing ExistingTags, newLoc LocationInfo) CleanedLocationTags {
	// 1. 构造当前最新地理分层零件
	var newHierarchyParts []string
	if newLoc.Country != "" {
		newHierarchyParts = append(newHierarchyParts, newLoc.Country)
	}
	if newLoc.Province != "" {
		newHierarchyParts = append(newHierarchyParts, newLoc.Province)
	}
	if newLoc.City != "" && newLoc.City != newLoc.Province {
		newHierarchyParts = append(newHierarchyParts, newLoc.City)
	}
	if newLoc.District != "" && newLoc.District != newLoc.City {
		newHierarchyParts = append(newHierarchyParts, newLoc.District)
	}

	var newTree string
	if len(newHierarchyParts) > 0 {
		newTree = strings.Join(newHierarchyParts, "|")
	}

	// 2. 收集旧地理树及其拆解出的原子地理词，用于精准清洗
	oldGeoWords := make(map[string]bool)
	oldGeoWords["中国"] = true
	oldGeoWords["China"] = true

	var retainedTrees []string
	for _, tree := range existing.HierarchicalSubject {
		cleanTree := strings.TrimSpace(tree)
		if cleanTree == "" {
			continue
		}

		// 判断是否属于地理树（如以 "中国|"、"China|" 开头，或者包含当前国家名）
		isGeoTree := strings.HasPrefix(cleanTree, "中国|") ||
			strings.HasPrefix(cleanTree, "China|") ||
			(newLoc.Country != "" && strings.HasPrefix(cleanTree, newLoc.Country+"|"))

		if isGeoTree {
			// 记录旧地理树中的每一个单词，作为清洗目标
			parts := strings.Split(cleanTree, "|")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					oldGeoWords[p] = true
				}
			}
		} else {
			// 非地理标签树（如 "题材|人像|户外"），完整保留
			if !slices.Contains(retainedTrees, cleanTree) {
				retainedTrees = append(retainedTrees, cleanTree)
			}
		}
	}

	// 合并最新的地理树
	if newTree != "" && !slices.Contains(retainedTrees, newTree) {
		retainedTrees = append(retainedTrees, newTree)
	}

	// 3. 清洗并合并平铺关键词 (Subject 与 Keywords)
	cleanKeywords := func(rawList []string) []string {
		var result []string
		seen := make(map[string]bool)

		for _, item := range rawList {
			cleanItem := strings.TrimSpace(item)
			if cleanItem == "" {
				continue
			}
			// 如果是旧地理词，剔除；否则保留用户自定义标签
			if oldGeoWords[cleanItem] {
				continue
			}
			if !seen[cleanItem] {
				seen[cleanItem] = true
				result = append(result, cleanItem)
			}
		}

		// 追加新的地理词
		for _, p := range newHierarchyParts {
			if !seen[p] {
				seen[p] = true
				result = append(result, p)
			}
		}
		return result
	}

	mergedSubject := cleanKeywords(existing.Subject)
	mergedKeywords := cleanKeywords(existing.Keywords)

	return CleanedLocationTags{
		HierarchicalSubject: retainedTrees,
		Subject:             mergedSubject,
		Keywords:            mergedKeywords,
	}
}
