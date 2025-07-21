//go:build !unix

package vmi_internal

func GetSysClktck() (int64, error) {
	return 100, nil
}
