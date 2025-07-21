//go:build unix

package vmi_internal

import (
	"github.com/tklauser/go-sysconf"
)

func GetSysClktck() (int64, error) {
	return sysconf.Sysconf(sysconf.SC_CLK_TCK)
}
