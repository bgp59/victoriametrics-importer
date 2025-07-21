// Command line arguments for the all VMIs

package vmi_internal

import (
	"bytes"
	"strings"
)

const (
	// The help usage message line wraparound default width:
	DEFAULT_FLAG_USAGE_WIDTH = 58
)

// Format command flag usage for help message, by wrapping the lines around a
// given width. The original line breaks and prefixing white spaces are ignored.
// Example:
//
// var  flagArg = flag.String(
//
//	name,
//	value,
//	FormatFlagUsageWidth(`
//	This usage message will be reformatted to the given width, discarding
//	the current line breaks and line prefixing spaces.
//	`, 40),
//
// )
func FormatFlagUsageWidth(usage string, width int) string {
	buf := &bytes.Buffer{}
	lineLen := 0
	for i, word := range strings.Fields(strings.TrimSpace(usage)) {
		// Perform line length checking only if this is not the 1st word:
		if i > 0 {
			if lineLen+len(word)+1 > width {
				buf.WriteByte('\n')
				lineLen = 0
			} else {
				buf.WriteByte(' ')
				lineLen++
			}
		}
		n, err := buf.WriteString(word)
		if err != nil {
			return usage
		}
		lineLen += n
	}
	return buf.String()
}

func FormatFlagUsage(usage string) string {
	return FormatFlagUsageWidth(usage, DEFAULT_FLAG_USAGE_WIDTH)
}
