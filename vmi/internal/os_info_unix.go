//go:build unix

package vmi_internal

import (
	"bytes"
	"fmt"

	"golang.org/x/sys/unix"
)

func GetOsInfo() (map[string]string, error) {
	zeroSuffixBufToString := func(buf []byte) string {
		i := bytes.IndexByte(buf, 0)
		if i < 0 {
			i = len(buf)
		}
		return string(buf[:i])
	}

	uname := unix.Utsname{}
	err := unix.Uname(&uname)
	if err != nil {
		return nil, fmt.Errorf("unix.Uname(): %v", err)
	}

	osInfo := make(map[string]string)
	osInfo["name"] = zeroSuffixBufToString(uname.Sysname[:])

	osRelease := zeroSuffixBufToString(uname.Release[:])
	osInfo["release"] = osRelease // e.g. 5.4.0-42-generic
	semVer := ""
	for _, c := range osRelease {
		if c != '.' && (c < '0' || '9' < c) {
			break
		}
		semVer += string(c)
	}
	osInfo["version"] = semVer
	osInfo["machine"] = zeroSuffixBufToString(uname.Machine[:])
	return osInfo, nil
}
