// Count available CPUs based on affinity

//go:build linux

package vmi_internal

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

// For linux count available CPUs based on CPU affinity, w/ a fallback on runtime:
func GetAvailableCPUCount() int {
	cpuSet := unix.CPUSet{}
	err := unix.SchedGetaffinity(os.Getpid(), &cpuSet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unix.SchedGetaffinity: %v", err)
		return runtime.NumCPU()
	}
	count := 0
	for _, cpuMask := range cpuSet {
		for cpuMask != 0 {
			count++
			cpuMask &= (cpuMask - 1)
		}
	}
	if count > runtime.NumCPU() {
		count = runtime.NumCPU()
	}
	return count
}
