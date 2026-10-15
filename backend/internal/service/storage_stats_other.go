//go:build !linux

package service

func getDiskSpace(path string) (total, free uint64, err error) {
	return 100 * 1024 * 1024 * 1024, 80 * 1024 * 1024 * 1024, nil
}
