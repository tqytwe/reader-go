package web

import "testing"

func TestChapterSimilarity(t *testing.T) {
	tests := []struct {
		a, b string
		min  float64
	}{
		{"第一章 开端", "第1章 开端", 0.85},
		{"第一百章 大结局", "大结局", 0.85},
		{"完全不同", "另一本书", 0},
	}
	for _, tc := range tests {
		got := chapterSimilarity(tc.a, tc.b)
		if got < tc.min {
			t.Errorf("chapterSimilarity(%q,%q)=%v want >= %v", tc.a, tc.b, got, tc.min)
		}
	}
}

func TestBestChapterMatch(t *testing.T) {
	titles := []string{"序章", "第一章 开始", "第二章 修炼"}
	score, idx := bestChapterMatch("第1章 开始", titles)
	if idx != 1 || score < 0.8 {
		t.Fatalf("bestChapterMatch got idx=%d score=%v", idx, score)
	}
}
