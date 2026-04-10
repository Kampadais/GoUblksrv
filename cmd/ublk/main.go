package main

import (
	"flag"
	"fmt"
	"os"

	ublk "github.com/Kampadais/GoUblksrv"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "del":
		delCmd(args)
	case "list":
		listCmd()
	case "info":
		infoCmd(args)
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("ublk-go: A CLI tool for ublk-lib")
	fmt.Println("\nUsage:")
	fmt.Println("  ublk del -n id")
	fmt.Println("  ublk list")
	fmt.Println("  ublk info -n id")
	fmt.Println("\nOptions for 'add':")
	fmt.Println("  -n int        Device ID to use (-1 for auto allocation) (default -1)")
}

func delCmd(args []string) {
	fs := flag.NewFlagSet("del", flag.ExitOnError)
	id := fs.Int("n", -1, "Device ID")
	fs.Parse(args)

	if *id == -1 {
		fmt.Println("Error: device ID is required")
		fs.Usage()
		os.Exit(1)
	}

	err := ublk.DeleteDevice(*id)
	if err != nil {
		fmt.Printf("Error deleting device: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully deleted device ublk%d\n", *id)
}

func listCmd() {
	devices := ublk.ListDevices()

	if len(devices) == 0 {
		fmt.Println("No ublk devices found.")
		return
	}

	fmt.Printf("%-5s %-10s %-10s %-15s %-10s\n", "ID", "Queues", "Depth", "Size", "Status")
	for _, dev := range devices {
		status := "Unknown"
		switch dev.State {
		case ublk.StateDead:
			status = "Dead"
		case ublk.StateLive:
			status = "Live"
		case ublk.StateQuiesced:
			status = "Quiesced"
		case ublk.StateFailIo:
			status = "FailIO"
		}
		fmt.Printf("%-5d %-10d %-10d %-15d %-10s\n", dev.ID, dev.NrHwQueues, dev.QueueDepth, dev.Size, status)
	}
}

func infoCmd(args []string) {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	id := fs.Int("n", -1, "Device ID")
	fs.Parse(args)

	if *id == -1 {
		fmt.Println("Error: device ID is required")
		fs.Usage()
		os.Exit(1)
	}

	info, err := ublk.GetDeviceInfo(*id)
	if err != nil {
		fmt.Printf("Error getting info for device %d: %v\n", *id, err)
		os.Exit(1)
	}

	fmt.Printf("Device Information for ublk%d:\n", info.ID)
	fmt.Printf("  ID:              %d\n", info.ID)
	fmt.Printf("  Queues:          %d\n", info.NrHwQueues)
	fmt.Printf("  Queue Depth:     %d\n", info.QueueDepth)
	fmt.Printf("  Size:            %d bytes\n", info.Size)
	fmt.Printf("  Block Size:      %d bytes\n", info.BlockSize)
	fmt.Printf("  Max IO Buf:      %d bytes\n", info.MaxIoBufBytes)
	fmt.Printf("  Ublksrv PID:     %d\n", info.UblksrvPid)
	fmt.Printf("  Owner UID/GID:   %d/%d\n", info.OwnerUID, info.OwnerGID)

	status := "Unknown"
	switch info.State {
	case ublk.StateDead:
		status = "Dead"
	case ublk.StateLive:
		status = "Live"
	case ublk.StateQuiesced:
		status = "Quiesced"
	case ublk.StateFailIo:
		status = "FailIO"
	}
	fmt.Printf("  State:           %s (%d)\n", status, info.State)
}
