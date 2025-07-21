//go:build !unix

package vmi_internal

import (
	"time"
)

func GetOsBootTime() (time.Time, error) {
	return time.Now(), nil
}
