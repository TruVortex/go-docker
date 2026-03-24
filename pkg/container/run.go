package container

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"go-docker/pkg/cgroups"
	"go-docker/pkg/fs"
)

func Run(cmd string, args []string) (err error) {
	
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in container.Run: %v", r)
		}
		if cleanupErr := cgroups.Cleanup(); cleanupErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; cgroup cleanup error: %v", err, cleanupErr)
			} else {
				err = fmt.Errorf("cgroup cleanup error: %w", cleanupErr)
			}
		}
	}()
	
	childArgs := append([]string{"child", cmd}, args...)
	
	c := exec.Command("/proc/self/exe", childArgs...)
	
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	
	c.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
	}
	
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	if err = c.Start(); err != nil {
		return fmt.Errorf("failed to start child process: %w", err)
	}

	childPID := c.Process.Pid

	if err = cgroups.SetLimits(childPID); err != nil {
		if errors.Is(err, cgroups.ErrCgroupUnavailable) {
			fmt.Fprintf(os.Stderr, "warning: cgroup unavailable, continuing without limits: %v\n", err)
		} else {
			_ = c.Process.Kill()
			return fmt.Errorf("failed to configure cgroups for pid %d: %w", childPID, err)
		}
	}
	
	signalForwardDone := make(chan struct{})
	go func() {
		defer close(signalForwardDone)
		for sig := range sigs {
			_ = c.Process.Signal(sig)
		}
	}()

	if err = c.Wait(); err != nil {
		return fmt.Errorf("child process exited with error: %w", err)
	}
	
	close(sigs)
	<-signalForwardDone

	return nil
}

func Child(cmd string, args []string) (err error) {
	
	rootfs := "/tmp/alpine-rootfs"

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in container.Child: %v", r)
		}
		if teardownErr := fs.Teardown(); teardownErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; fs teardown error: %v", err, teardownErr)
			} else {
				err = fmt.Errorf("fs teardown error: %w", teardownErr)
			}
		}
	}()

	if err = fs.Setup(rootfs); err != nil {
		return fmt.Errorf("fs setup (chroot+proc mount) failed: %w", err)
	}
	
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	go func() {
		for sig := range sigs {
			if s, ok := sig.(syscall.Signal); ok {
				_ = syscall.Kill(syscall.Getpid(), s)
			}
		}
	}()
	
	os.Setenv("PATH", "/bin:/usr/bin:/sbin:/usr/sbin")

	var resolvedPath string
	if filepath.IsAbs(cmd) {
		resolvedPath = cmd
	} else {
		resolvedPath, err = exec.LookPath(cmd)
		if err != nil {
			return fmt.Errorf("failed to resolve command %q in PATH: %w", cmd, err)
		}
	}

	argv := append([]string{cmd}, args...)

	if err = syscall.Exec(resolvedPath, argv, os.Environ()); err != nil {
		return fmt.Errorf("syscall.Exec %q failed: %w", resolvedPath, err)
	}

	return nil
}
