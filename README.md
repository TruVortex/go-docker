# Mini-Docker (Go)

Custom lightweight container runtime (namespace + chroot + cgroups) in pure Go (standard library only)

## Overview

`go-docker` demonstrates:

- `run` / `child` re-exec workflow
- Linux namespaces: PID, UTS, mount
- filesystem isolation via `chroot` (Alpine rootfs) + `/proc` mount
- cgroups v2 resource control (memory limit) with host fallback
- process replacement with `syscall.Exec` (PID 1 inside container)
- signal handling and cleanup (`SIGINT`, `SIGTERM`)

## Project Structure

- `main.go` - CLI bootstrap, run vs child command
- `pkg/container/run.go` - namespace spawn and lifecycle logic
- `pkg/fs/fs.go` - fs isolation (`chroot`, proc mount/unmount)
- `pkg/cgroups/cgroups.go` - memory cgroup setup and cleanup

## Prerequisites

- Linux host with namespaces and cgroup2
- Go 1.21+
- root privileges for namespace/cgroup operations
- Alpine rootfs at `/tmp/alpine-rootfs`

### Alpine rootfs setup

```bash
mkdir -p /tmp/alpine-rootfs
cd /tmp/alpine-rootfs
wget https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz
tar xf alpine-minirootfs-3.19.1-x86_64.tar.gz
rm alpine-minirootfs-3.19.1-x86_64.tar.gz
```

(Optional) install bash in Alpine:

```bash
sudo chroot /tmp/alpine-rootfs /bin/sh -c "apk update && apk add bash"
```

## Build & Run

```bash
cd /go-docker
go mod tidy
```

### Interactive container shell

```bash
sudo go run main.go run /bin/sh
```

### Run a single command

```bash
sudo go run main.go run /bin/sh -c 'id; hostname; echo yes'
sudo go run main.go run id
sudo go run main.go run /bin/echo done
sudo go run main.go run sh -c 'echo hi'
```

### Long-running command

```bash
sudo go run main.go run /bin/sh -c 'while true; do echo hi; sleep 1; done'
```

### Exit container

From container:
- `exit`
- `Ctrl-D`

From host (if needed):

- `sudo pkill -f 'main.go run'`

## Cgroup Behavior

- On some hosts, `/sys/fs/cgroup` may be read-only (e.g. `dr-xr-xr-x`).
- Then runtime logs warning and continues without memory limits.
- In writable mode, it apps:
  - `/sys/fs/cgroup/mini-docker/memory.max` = `50M`
  - `/sys/fs/cgroup/mini-docker/cgroup.procs` = child pid

## Test commands

```bash
sudo go run main.go run /bin/ps -ef
sudo go run main.go run /bin/cat /etc/os-release
sudo go run main.go run /bin/hostname
sudo go run main.go run /bin/ls /proc
```

## Confirm containerization

- PID 1 inside container (`ps` view limited)
- `cat /etc/os-release` from Alpine filesystem
- `chroot` isolation active
- `syscall.Exec` makes target the PID 1 process

## Troubleshooting

- `bash` may be missing in Alpine rootfs: use `sh` or install bash with `apk add bash`.
- Cgroup unavailable warning means host cgroup root not writable; use a suitable host/VM.
- Must run as root for namespace creation and cgroups.