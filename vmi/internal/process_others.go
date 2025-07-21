//go:build !unix

package vmi_internal

func GetCpuTime(who int) (float64, error) {
	return -1, nil
}

func GetMyCpuTime() (float64, error) {
	return -1, nil
}
