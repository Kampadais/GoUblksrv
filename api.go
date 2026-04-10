package GoUblksrv

import (
	"fmt"
	"os"
)

type UblkParams struct {
	Queues     int
	QueueDepth int
	Size       int64
	BlockSize  int
}

func NewUblkDevice(name string, params UblkParams) *UblkDevice {
	if params.BlockSize == 0 {
		params.BlockSize = 4096
	}
	d := &UblkDevice{
		Volume:     name,
		ID:         -1,
		Queues:     params.Queues,
		QueueDepth: params.QueueDepth,
		Size:       params.Size,
		BlockSize:  params.BlockSize,
		requests:   make(chan *Request, 4096),
		msgChan:    make(chan *Request, 4096),
		done:       make(chan struct{}),
		daemonDone: make(chan struct{}),
	}
	return d
}

func ListDevices() []DeviceInfo {
	var devices []DeviceInfo
	entries, err := os.ReadDir("/sys/class/ublk")
	if err != nil {
		// Fallback to scanning 0-63 if /sys/class/ublk is not available
		for i := 0; i < 64; i++ {
			info, err := getDeviceInfo(i)
			if err == nil {
				devices = append(devices, *info)
			}
		}
		return devices
	}

	for _, entry := range entries {
		var id int
		_, err := fmt.Sscanf(entry.Name(), "ublkc%d", &id)
		if err == nil {
			info, err := getDeviceInfo(id)
			if err == nil {
				devices = append(devices, *info)
			}
		}
	}
	return devices
}

func GetDeviceInfo(id int) (*DeviceInfo, error) {
	return getDeviceInfo(id)
}

func (d *UblkDevice) Start(handler IOHandler) error {
	d.handler = handler

	// Pre-allocate messages
	for i := range 512 {
		msg := &Request{
			Complete:     make(chan struct{}, 1),
			MagicVersion: MagicVersion,
			Seq:          uint32(i),
			Type:         uint32(100),
			RData:        make([]byte, Blocks*1024),
		}
		d.msgChan <- msg
	}

	go d.handleRequests()

	return d.startupC()
}

func (d *UblkDevice) Delete() error {
	return DeleteDevice(d.ID)
}

func DeleteDevice(id int) error {
	d := &UblkDevice{
		ID:         id,
		daemonDone: make(chan struct{}),
	}

	return d.deleteC()
}

func (d *UblkDevice) GetInfo() (*DeviceInfo, error) {
	return getDeviceInfo(d.ID)
}
