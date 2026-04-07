# ublk-lib Examples

This directory contains examples of how to use the `ublk-lib` library.

## File-backed ublk device

This example creates a ublk device that uses a file on your local filesystem as a backing store.

### Prerequisites

- Linux kernel with `ublk` support (usually 6.0+).
- `liburing` installed on your system.
- `ublk` kernel module loaded: `sudo modprobe ublk_drv`

### Running the example

To run the example, you need root privileges to manage ublk devices.

```bash
# From the root of the project
sudo go run ./examples/file-backed/main.go -file my-disk.img -size 100000000
```

This will:
1. Create a 100MB file named `my-disk.img` (if it doesn't exist).
2. Create a ublk device (e.g., `/dev/ublkb0`).
3. Handle all IO requests to that device by reading/writing from the file.

While it is running, you can format and mount the device:

```bash
sudo mkfs.ext4 /dev/ublkb0
sudo mount /dev/ublkb0 /mnt
# Try writing some files to /mnt
```

Press `Ctrl+C` to gracefully stop and remove the device.
