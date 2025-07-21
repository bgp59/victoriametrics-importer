// Misc Unix OS related info

//go:build unix

package vmi_internal

import (
	"bufio"
	"os"

	"bytes"
	"fmt"
	"strings"
)

func parseOsReleaseFile(filePath string) (map[string]string, error) {
	osRelease := make(map[string]string)
	fh, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		line := scanner.Text()
		i := strings.Index(line, "=")
		if i <= 1 || i == len(line)-1 {
			continue
		}
		key, val := strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
		if len(key) == 0 {
			continue
		}
		valLen := len(val)
		if valLen > 0 && val[0] == '"' {
			val = val[1:]
			valLen -= 1
		}
		if valLen > 0 && val[valLen-1] == '"' {
			val = val[:valLen-1]
			valLen -= 1
		}
		if valLen == 0 {
			continue
		}
		osRelease[strings.ToLower(key)] = val
	}
	fh.Close()
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	return osRelease, nil
}

func GetOsReleaseInfo() (map[string]string, error) {
	errBuf := &bytes.Buffer{}

	for _, filePath := range []string{
		"/etc/os-release",
		"/usr/lib/os-release",
	} {
		if osRelease, err := parseOsReleaseFile(filePath); err == nil {
			return osRelease, nil
		} else {
			if errBuf.Len() > 0 {
				errBuf.WriteString(", ")
			}
			fmt.Fprintf(errBuf, "%v", err)
		}
	}
	return nil, fmt.Errorf("%s", errBuf.String())
}
