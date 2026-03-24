package fs

import (
	"fmt"
	"syscall"
)

func Setup(rootfsPath string) error {
	if err := syscall.Chroot(rootfsPath); err != nil {
		return fmt.Errorf("chroot %s failed: %w", rootfsPath, err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / failed: %w", err)
	}
	if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount proc failed: %w", err)
	}
	return nil
}

func Teardown() error {
	if err := syscall.Unmount("/proc", 0); err != nil {
		return fmt.Errorf("unmount /proc failed: %w", err)
	}
	return nil
}
