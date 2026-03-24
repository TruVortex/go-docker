package cgroups

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	defaultCgroupRoot = "/sys/fs/cgroup/mini-docker"
	memoryMax         = "50M"
)

var (
	cgroupRoot            string
	ErrCgroupUnavailable = errors.New("cgroup not available or permission denied")
)

func init() {
	cgroupRoot = chooseCgroupRoot()
}

func chooseCgroupRoot() string {
	candidates := []string{
		"/sys/fs/cgroup/mini-docker",
		"/sys/fs/cgroup/system.slice/mini-docker",
		"/sys/fs/cgroup/user.slice/user-0.slice/mini-docker",
		"/sys/fs/cgroup/user.slice/user-1000.slice/mini-docker",
	}
	for _, cand := range candidates {
		parent := filepath.Dir(cand)
		if dirWritable(parent) {
			return cand
		}
	}
	return defaultCgroupRoot
}

func dirWritable(path string) bool {
	if path == "" {
		return false
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return false
	}
	testFile := filepath.Join(path, ".cgroup-write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(testFile)
	return true
}

func SetLimits(pid int) error {
	if !dirWritable(filepath.Dir(cgroupRoot)) {
		return fmt.Errorf("%w: %q parent is not writable", ErrCgroupUnavailable, cgroupRoot)
	}

	if err := os.MkdirAll(cgroupRoot, 0755); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("%w: could not mkdir %q: %w", ErrCgroupUnavailable, cgroupRoot, err)
		}
		return fmt.Errorf("failed to create cgroup dir %q: %w", cgroupRoot, err)
	}

	memPath := filepath.Join(cgroupRoot, "memory.max")
	if err := os.WriteFile(memPath, []byte(memoryMax), 0644); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("%w: could not write %q: %w", ErrCgroupUnavailable, memPath, err)
		}
		return fmt.Errorf("failed to write %q: %w", memPath, err)
	}

	procsPath := filepath.Join(cgroupRoot, "cgroup.procs")
	if err := os.WriteFile(procsPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("%w: could not write %q: %w", ErrCgroupUnavailable, procsPath, err)
		}
		return fmt.Errorf("failed to write %q: %w", procsPath, err)
	}

	return nil
}

func Cleanup() error {
	if cgroupRoot == "" {
		return nil
	}
	if err := os.RemoveAll(cgroupRoot); err != nil {
		return fmt.Errorf("failed to remove cgroup tree %q: %w", cgroupRoot, err)
	}
	return nil
}
