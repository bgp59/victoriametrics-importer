//go:build !unix

package vmi_internal

func GetOSInfo() (map[string]string, error) {
	return map[string]string{}, nil
}
