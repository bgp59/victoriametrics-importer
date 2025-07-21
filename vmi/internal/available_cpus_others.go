// Count available CPUs based on affinity

//go:build !linux

package vmi_internal

import (
	"runtime"
)

func GetAvailableCPUCount() int {
	return runtime.NumCPU()
}
