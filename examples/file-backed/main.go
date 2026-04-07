package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	ublk "github.com/Kampadais/GoUblksrv"
)

type FileHandler struct {
	file *os.File
}

func (h *FileHandler) ReadAt(p []byte, off int64) (int, error) {
	return h.file.ReadAt(p, off)
}

func (h *FileHandler) WriteAt(p []byte, off int64) (int, error) {
	return h.file.WriteAt(p, off)
}

func (h *FileHandler) UnmapAt(length uint32, off int64) (int, error) {
	// For a simple file, we can just return success.
	return int(length), nil
}

func main() {
	filePath := flag.String("file", "test.img", "Path to the backing file")
	size := flag.Int64("size", 1024*1024*100, "Device size in bytes (default 100MB)")
	flag.Parse()

	// Ensure backing file exists
	f, err := os.OpenFile(*filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	// Truncate to size if necessary
	if err := f.Truncate(*size); err != nil {
		log.Fatalf("Failed to truncate file: %v", err)
	}

	params := ublk.UblkParams{
		Queues:     1,
		QueueDepth: 64,
		Size:       *size,
		BlockSize:  4096,
	}

	dev, err := ublk.NewUblkDevice("example-file", params)
	if err != nil {
		log.Fatalf("Failed to create ublk device: %v", err)
	}

	handler := &FileHandler{file: f}

	//Starting the Device
	if err := dev.Start(handler); err != nil {
		log.Fatalf("Failed to start ublk device: %v", err)
	}

	fmt.Printf("ublk device %d started. You can use it at /dev/ublkb%d\n", dev.ID, dev.ID)

	// Wait for the device to be started with timeout
	timeout := time.Now().Add(2 * time.Second)
	for time.Now().Before(timeout) {
		info, err := dev.GetInfo()
		if err != nil {
			log.Printf("Failed to get device info: %v", err)
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if info.UblksrvPid != 0 {
			fmt.Printf("Device info: ID=%d, State=%d, UblksrvPid=%d\n", info.ID, info.State, info.UblksrvPid)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println("Press Ctrl+C to stop")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nStopping...")

	dev.Delete()

}
