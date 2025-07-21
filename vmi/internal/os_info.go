package vmi_internal

import (
	"fmt"
	"os"
	"time"
)

var (
	AvailableCPUCount = GetAvailableCPUCount()
	BootTime          = time.Now()
	Clktck            int64
	ClktckSec         float64
	OsInfo            = make(map[string]string)
	OsRelease         = make(map[string]string)
)

func init() {
	bootTime, err := GetOsBootTime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetOsBootTime(): %v\n", err)
	} else {
		BootTime = bootTime
	}

	clktck, err := GetSysClktck()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetSysClktck(): %v\n", err)
	} else {
		Clktck = clktck
		ClktckSec = float64(1) / float64(Clktck)
	}

	osInfo, err := GetOsInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetOSInfo: %v\n", err)
	} else {
		OsInfo = osInfo
	}

	osRelease, err := GetOsReleaseInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetOsReleaseInfo(): %v\n", err)
	} else {
		OsRelease = osRelease
	}
}
