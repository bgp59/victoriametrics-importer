//go:build unix

package vmi_internal

import (
	"golang.org/x/sys/unix"
)

func GetCpuTime(who int) (float64, error) {
	rusage := &unix.Rusage{}
	err := unix.Getrusage(who, rusage)
	if err != nil {
		return 0, err
	}
	return (float64(rusage.Utime.Sec+rusage.Stime.Sec) +
		float64(rusage.Utime.Usec+rusage.Stime.Usec)/1e6), nil
}

func GetMyCpuTime() (float64, error) {
	return GetCpuTime(unix.RUSAGE_SELF)
}
