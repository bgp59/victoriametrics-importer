//go:build !linux

package vmi_internal

func GetOsReleaseInfo() (map[string]string, error) {
	return map[string]string{}, nil
}
