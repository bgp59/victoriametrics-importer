//go:build unix

package vmi_internal

import (
	"fmt"
	"time"

	"github.com/mackerelio/go-osstat/uptime"
)

func GetOsBootTime() (time.Time, error) {
	uptime, err := uptime.Get()
	if err != nil {
		return time.Now(), fmt.Errorf("uptime.Get(): %v", err)
	}
	return time.Now().Add(-uptime), nil
}
