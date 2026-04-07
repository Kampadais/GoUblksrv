# Goublksrv

A standalone Go library for building `ublk` (Userspace Block Device) targets in Linux.

This library is a golang wrapper of the [ublksrv](https://github.com/ublk-org/ublksrv) C library, providing a high-level API for creating and managing `ublk` devices allowing developers to focus on implementing their block device logic in Golang.

Most of the code is copied from the original C library, with some adjustments regarding C++ functionallities reimplemented in C and adapted to Go's cgo.

The library also includes a CLI tool like the original C library.
## Prerequisites

- **Linux Kernel**: 6.0 or newer with `ublk` support enabled.
- **Kernel Module**: `sudo modprobe ublk_drv`
- **Dependencies**: `liburing-dev` must be installed on your system.
- **Go**: 1.25 or newer.

## Project Structure

- `cmd/ublk/`: A CLI tool for managing `ublk` devices (add, del, list, info).
- `examples/`: Ready-to-use examples demonstrating library usage.
  - `file-backed/`: A simple implementation that uses a local file as the backing store for a block device.

## Getting Started

To use the library in your project, implement the `IOHandler` interface:

```go
type IOHandler interface {
	ReadAt(p []byte, off int64) (int, error)
	WriteAt(p []byte, off int64) (int, error)
	UnmapAt(length uint32, off int64) (int, error)
}
```

Then create and start a device:

```go
params := ublk.UblkParams{
    Queues:     2,
    QueueDepth: 128,
    Size:       1024 * 1024 * 1024, // 1GB
}

dev, _ := ublk.NewUblkDevice("my-device", params)
_ = dev.Start(myHandler)
defer dev.Delete()
```

## Advanced API

The library also provides static methods for device management:

```go
// List all ublk devices on the system
devices, _ := ublk.ListDevices()

// Get information about a specific device
info, _ := ublk.GetDeviceInfo('ID')

// Delete a device by ID
_ = ublk.DeleteDevice('ID')

// Add a device without starting a daemon
id, _ := ublk.AddDevice(params)
```

## CLI Tool

The project includes a command-line tool for managing `ublk` devices.

To build the CLI:
```bash
cd cmd/ublk
go build -o ublk-go .
```

Usage:
```bash
# List all ublk devices
./ublk-go list

# Get info for a specific device
./ublk-go info -n 0

# Delete a device
sudo ./ublk-go del -n 0
```

## Running the Example

The `file-backed` example creates a virtual block device backed by a local file.

```bash
# Clone and enter the project
# Ensure you have root privileges for ublk management
sudo go run ./examples/file-backed/main.go -file my-disk.img -size 100000000
```

For more detailed instructions on running the examples, see the [Examples README](examples/README.md).

## Development

This project uses a Go workspace (`go.work`) to manage the library and examples locally.

To build the library:
```bash
cd ublk-lib
go build .
```

