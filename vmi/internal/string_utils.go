package vmi_internal

import (
	"regexp"
	"strings"
)

var wordSplitRe = regexp.MustCompile(`\s+`)

func SplitWords(s string) []string {
	return wordSplitRe.Split(strings.TrimSpace(s), -1)
}
