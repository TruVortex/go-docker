package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"go-docker/pkg/container"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <run|child> <cmd> [args...]\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	done := make(chan error, 1)
	go func() {
		done <- runCommand(os.Args[1], os.Args[2:])
	}()

	select {
	case sig := <-sigs:
		fmt.Fprintf(os.Stderr, "received signal %s, attempting graceful shutdown...\n", sig)
		if sig == syscall.SIGINT || sig == syscall.SIGTERM {
			
		}
		return
	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
			os.Exit(1)
		}
	}
}

func runCommand(mode string, args []string) error {
	switch mode {
	case "run":
		if len(args) < 1 {
			return fmt.Errorf("usage: run <cmd> [args...]")
		}
		return container.Run(args[0], args[1:])

	case "child":
		if len(args) < 1 {
			return fmt.Errorf("usage: child <cmd> [args...]")
		}
		return container.Child(args[0], args[1:])

	default:
		return fmt.Errorf("unknown command: %s", mode)
	}
}
