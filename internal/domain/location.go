package domain

import (
	"github.com/vincentchyu/photools/pkg/geocoding"
)

// LocationInfo 逆地理编码返回的地理位置详细信息 (映射到 pkg/geocoding.LocationInfo)
type LocationInfo = geocoding.LocationInfo

var (
	ToAlpha3 = geocoding.ToAlpha3
	ToAlpha2 = geocoding.ToAlpha2
)
