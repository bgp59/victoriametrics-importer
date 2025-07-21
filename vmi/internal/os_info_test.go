package vmi_internal

import (
	"bytes"
	"fmt"
	"sort"
	"testing"
)

func TestOsAvailableCPUCount(t *testing.T) {
	t.Logf("GetAvailableCPUCount() = %d", AvailableCPUCount)
}

func TestOsSysClktck(t *testing.T) {
	t.Logf("Clktck = %d, ClktckSec = %.06f", Clktck, ClktckSec)
}

func TestOsBootTime(t *testing.T) {
	t.Logf("BootTime = %s", BootTime)
}

func TestOsInfo(t *testing.T) {
	buf := new(bytes.Buffer)
	keys := []string{}
	for key := range OsInfo {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(buf, "\t%s: %q\n", key, OsInfo[key])
	}
	t.Logf("OsInfo:\n%s", buf)
}

func TestOsRelease(t *testing.T) {
	buf := new(bytes.Buffer)
	keys := []string{}
	for key := range OsRelease {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(buf, "\t%s: %q\n", key, OsRelease[key])
	}
	t.Logf("OsRelease:\n%s", buf)
}

func TestOsGetMyCpuTime(t *testing.T) {
	cpuTime, err := GetMyCpuTime()
	if err != nil {
		t.Fatalf("GetCpuTime(): %v", err)
	}
	t.Logf("GetCpuTime() = %f", cpuTime)
}
