package web

import (
	"regexp"
	"strings"
	"unicode"
)

var chapterPrefixRe = regexp.MustCompile(`^(第[\d一二三四五六七八九十百千万]+章[：:\s\-—]*)`)

// normalizeChapterTitle 归一化章节名便于对齐
func normalizeChapterTitle(s string) string {
	s = strings.TrimSpace(s)
	s = chapterPrefixRe.ReplaceAllString(s, "")
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '　' {
			return -1
		}
		return unicode.ToLower(r)
	}, s)
	return s
}

// chapterSimilarity 返回 0~1，1 表示完全匹配
func chapterSimilarity(a, b string) float64 {
	na, nb := normalizeChapterTitle(a), normalizeChapterTitle(b)
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1
	}
	if strings.Contains(na, nb) || strings.Contains(nb, na) {
		return 0.88
	}
	// 字符集合 Jaccard
	setA := runeSet(na)
	setB := runeSet(nb)
	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}
	inter := 0
	for r := range setA {
		if setB[r] {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func runeSet(s string) map[rune]bool {
	m := make(map[rune]bool)
	for _, r := range s {
		m[r] = true
	}
	return m
}

// bestChapterMatch 在目录中找与 current 最相似的章节
func bestChapterMatch(current string, titles []string) (score float64, index int) {
	if strings.TrimSpace(current) == "" || len(titles) == 0 {
		return 0, -1
	}
	bestScore := 0.0
	bestIdx := -1
	for i, t := range titles {
		s := chapterSimilarity(current, t)
		if s > bestScore {
			bestScore = s
			bestIdx = i
		}
	}
	return bestScore, bestIdx
}

// bookMatchScore 书名+作者综合匹配分
func bookMatchScore(targetName, targetAuthor, name, author string) float64 {
	nameScore := textSimilarity(targetName, name)
	authorScore := textSimilarity(targetAuthor, author)
	if strings.TrimSpace(targetAuthor) == "" {
		return nameScore
	}
	return nameScore*0.65 + authorScore*0.35
}

func textSimilarity(a, b string) float64 {
	a = strings.TrimSpace(strings.ToLower(a))
	b = strings.TrimSpace(strings.ToLower(b))
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 0.9
	}
	return chapterSimilarity(a, b)
}
