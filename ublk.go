package GoUblksrv

import (
	"fmt"
)

type UblkDevice struct {
	ID         int
	Volume     string
	Size       int64
	Queues     int
	QueueDepth int
	BlockSize  int
	DaemonPId  int

	requests   chan *Request
	msgChan    chan *Request
	done       chan struct{}
	daemonDone chan struct{}
	handler    IOHandler

	isUp bool
}

func (d *UblkDevice) handleRequests() {
	for {
		select {
		case msg := <-d.requests:
			switch msg.Type {
			case TypeRead:
				go d.handleRead(msg)
			case TypeWrite:
				go d.handleWrite(msg)
			case TypeUnmap:
				go d.handleUnmap(msg)
			case TypePing:
				go d.handlePing(msg)
			default:
				fmt.Printf("Unknown message type: %d\n", msg.Type)
				msg.Complete <- struct{}{}
			}
		case <-d.done:
			return
		}
	}
}

func (d *UblkDevice) handleRead(msg *Request) {
	msg.RData = msg.RData[:msg.Size]
	_, err := d.handler.ReadAt(msg.RData, msg.Offset)
	if err != nil {
		fmt.Printf("Read error: %v\n", err)
	}
	msg.Complete <- struct{}{}
}

func (d *UblkDevice) handleWrite(msg *Request) {
	_, err := d.handler.WriteAt(msg.WData, msg.Offset)
	if err != nil {
		fmt.Printf("Write error: %v\n", err)
	}
	msg.Complete <- struct{}{}
}

func (d *UblkDevice) handleUnmap(msg *Request) {
	_, err := d.handler.UnmapAt(msg.Size, msg.Offset)
	if err != nil {
		fmt.Printf("Unmap error: %v\n", err)
	}
	msg.Complete <- struct{}{}
}

func (d *UblkDevice) handlePing(msg *Request) {
	msg.Complete <- struct{}{}
}
