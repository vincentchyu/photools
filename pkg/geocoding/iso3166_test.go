package geocoding

import (
	"testing"
)

func TestISO3166Conversion(t *testing.T) {
	tests := []struct {
		alpha2 string
		alpha3 string
	}{
		{"CN", "CHN"},
		{"cn", "CHN"},
		{"US", "USA"},
		{"us", "USA"},
		{"JP", "JPN"},
		{"HK", "HKG"},
		{"MO", "MAC"},
		{"TW", "TWN"},
		{"GB", "GBR"},
		{"FR", "FRA"},
		{"DE", "DEU"},
		{"IT", "ITA"},
		{"RU", "RUS"},
		{"KR", "KOR"},
		{"CA", "CAN"},
		{"AU", "AUS"},
		{"NZ", "NZL"},
		{"SG", "SGP"},
	}

	for _, tt := range tests {
		gotA3 := ToAlpha3(tt.alpha2)
		if gotA3 != tt.alpha3 {
			t.Errorf("ToAlpha3(%q) = %q; want %q", tt.alpha2, gotA3, tt.alpha3)
		}

		gotA2 := ToAlpha2(tt.alpha3)
		wantA2 := ToAlpha2(tt.alpha2)
		if gotA2 != wantA2 {
			t.Errorf("ToAlpha2(%q) = %q; want %q", tt.alpha3, gotA2, wantA2)
		}
	}
}

func TestISO3166EdgeCases(t *testing.T) {
	t.Run("已经是三字码", func(t *testing.T) {
		if got := ToAlpha3("CHN"); got != "CHN" {
			t.Errorf("ToAlpha3(CHN) = %q, want CHN", got)
		}
		if got := ToAlpha3("chn"); got != "CHN" {
			t.Errorf("ToAlpha3(chn) = %q, want CHN", got)
		}
	})

	t.Run("已经是二字码", func(t *testing.T) {
		if got := ToAlpha2("CN"); got != "CN" {
			t.Errorf("ToAlpha2(CN) = %q, want CN", got)
		}
		if got := ToAlpha2("cn"); got != "CN" {
			t.Errorf("ToAlpha2(cn) = %q, want CN", got)
		}
	})

	t.Run("未知代码与空值", func(t *testing.T) {
		if got := ToAlpha3(""); got != "" {
			t.Errorf("ToAlpha3(\"\") = %q, want empty", got)
		}
		if got := ToAlpha2(""); got != "" {
			t.Errorf("ToAlpha2(\"\") = %q, want empty", got)
		}
		if got := ToAlpha3("ZZ"); got != "ZZ" {
			t.Errorf("ToAlpha3(ZZ) = %q, want ZZ", got)
		}
		if got := ToAlpha2("ZZZ"); got != "ZZZ" {
			t.Errorf("ToAlpha2(ZZZ) = %q, want ZZZ", got)
		}
	})
}
