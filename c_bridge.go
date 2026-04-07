package GoUblksrv

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -luring
#include "ublkhelper.h"
#include <stdlib.h>

extern void onRequestAsync(struct msghdr *msg, struct message *req, int opType, struct ublksrv_queue *q, struct ublk_io_data *data);
*/
import "C"
import (
	"fmt"
	"os"
	"sync"
	"unsafe"
)

var Done = make(chan struct{})
var (
	devicesMu sync.RWMutex
	devices   = make(map[int]*UblkDevice)
)

func registerDevice(id int, d *UblkDevice) {
	devicesMu.Lock()
	devices[id] = d
	devicesMu.Unlock()
}

func unregisterDevice(id int) {
	devicesMu.Lock()
	delete(devices, id)
	devicesMu.Unlock()
}

//export onRequestAsync
func onRequestAsync(msg *C.struct_msghdr, req *C.struct_message, opType C.int, q *C.struct_ublksrv_queue, data *C.struct_ublk_io_data) {
	devID := int(C.ublksrv_get_ctrl_dev(q.dev).dev_info.dev_id)
	devicesMu.RLock()
	d := devices[devID]
	devicesMu.RUnlock() // TODO Check this

	if d == nil {
		return
	}

	iovecs := (*[2]C.struct_iovec)(unsafe.Pointer(msg.msg_iov))[:msg.msg_iovlen:msg.msg_iovlen]
	dataPtr := iovecs[1].iov_base
	dataLen := iovecs[1].iov_len

	reqMsg := <-d.msgChan

	reqMsg.Size = uint32(req.size)
	reqMsg.Seq = uint32(C.int(req.seq))
	reqMsg.Type = uint32(opType)
	reqMsg.Offset = int64(req.offset)

	if opType == C.LONGHORN_CMD_TYPE_WRITE {
		reqMsg.WData = unsafe.Slice((*byte)(dataPtr), dataLen)
	}

	go func(msgObj *Request, opType C.int, dataPtr unsafe.Pointer, dataLen C.size_t, q *C.struct_ublksrv_queue, data *C.struct_ublk_io_data) {
		d.requests <- msgObj
		<-msgObj.Complete

		if opType == C.LONGHORN_CMD_TYPE_READ {
			dst := unsafe.Slice((*byte)(dataPtr), dataLen)
			copy(dst, msgObj.RData)
		}

		if opType == C.LONGHORN_CMD_TYPE_UNMAP {
			C.ublksrv_complete_io(q, C.uint(data.tag), 0)
		} else {
			nrSectors := C.get_nr_sectors(data.iod)
			C.ublksrv_complete_io(q, C.uint(data.tag), C.int(nrSectors<<9))
		}
		d.msgChan <- msgObj
	}(reqMsg, opType, dataPtr, dataLen, q, data)
}

//export notifyShutdown
func notifyShutdown() {
	Done <- struct{}{}

}

func (d *UblkDevice) addC() error {
	fmt.Println("ADDC()")
	fmt.Println("Adding device with ID:", d.ID)
	nrHwQueues := d.Queues
	if nrHwQueues <= 0 {
		nrHwQueues = int(C.DEF_NR_HW_QUEUES)
	}

	queueDepth := d.QueueDepth
	if queueDepth <= 0 {
		queueDepth = int(C.DEF_QD)
	}

	runDir := C.UBLKSRV_PID_DIR

	data := C.struct_ublksrv_dev_data{
		queue_depth:      C.ushort(queueDepth),
		nr_hw_queues:     C.ushort(nrHwQueues),
		run_dir:          C.CString(runDir),
		max_io_buf_bytes: C.uint(C.DEF_BUF_SIZE),
	}
	d.isUp = true

	go func() {
		dev := C.ublksrv_ctrl_init(&data)
		C.ublksrv_ctrl_add_dev(dev)
		sectors := uint64(d.Size) >> 9
		C.init_params(dev, C.__u64(sectors))

		d.ID = int(dev.dev_info.dev_id)

		C.ublksrv_start_daemon(dev)
	}()

	return nil
}

func (d *UblkDevice) startupC() error {
	err := os.MkdirAll("/tmp/ublksrvd", 0755)
	if err != nil {
		return fmt.Errorf("error creating directory /tmp/ublksrvd: %w", err)
	}
	nrHwQueues := d.Queues
	if nrHwQueues <= 0 {
		nrHwQueues = int(C.DEF_NR_HW_QUEUES)
	}

	queueDepth := d.QueueDepth
	if queueDepth <= 0 {
		queueDepth = int(C.DEF_QD)
	}

	runDir := C.UBLKSRV_PID_DIR
	cRunDir := C.CString(runDir)
	fmt.Println("Using run dir:", runDir)
	// We don't free cRunDir here because it's used in a background goroutine

	data := C.struct_ublksrv_dev_data{
		queue_depth:      C.ushort(queueDepth),
		nr_hw_queues:     C.ushort(nrHwQueues),
		run_dir:          cRunDir,
		max_io_buf_bytes: C.uint(C.DEF_BUF_SIZE),
	}

	d.isUp = true

	initDone := make(chan error, 1)
	go func() {
		dev := C.ublksrv_ctrl_init(&data)
		C.ublksrv_ctrl_add_dev(dev)
		sectors := uint64(d.Size) >> 9
		C.init_params(dev, C.__u64(sectors))

		d.ID = int(dev.dev_info.dev_id)
		registerDevice(d.ID, d)
		initDone <- nil
		C.ublksrv_start_daemon(dev)
		unregisterDevice(d.ID)
		C.ublksrv_ctrl_deinit(dev)
		close(d.daemonDone)
	}()

	return <-initDone
}

func (d *UblkDevice) deleteC() {
	go C.cmd_dev_del(C.int(d.ID))
	<-Done
	err := os.Remove(fmt.Sprint(C.UBLKSRV_PID_DIR, "/", d.ID, ".pid"))
	if err != nil {
		fmt.Println("Error deleting PID file:", err)
		return
	}
}

func getDeviceInfo(id int) (*DeviceInfo, error) {
	data := C.struct_ublksrv_dev_data{
		dev_id: C.int(id),
	}
	dev := C.ublksrv_ctrl_init(&data)
	if dev == nil {
		return nil, fmt.Errorf("failed to init ctrl for dev %d", id)
	}
	defer C.ublksrv_ctrl_deinit(dev)

	ret := C.ublksrv_ctrl_get_info(dev)
	if ret < 0 {
		return nil, fmt.Errorf("failed to get info for dev %d: %d", id, int(ret))
	}

	var params C.struct_ublk_params
	ret = C.ublksrv_ctrl_get_params(dev, &params)
	if ret < 0 {
		// Some devices might not have params set yet if they just added
		// But let's at least return what we have
	}

	info := &DeviceInfo{
		ID:            int(dev.dev_info.dev_id),
		NrHwQueues:    int(dev.dev_info.nr_hw_queues),
		QueueDepth:    int(dev.dev_info.queue_depth),
		State:         int(dev.dev_info.state),
		MaxIoBufBytes: uint32(dev.dev_info.max_io_buf_bytes),
		UblksrvPid:    int(dev.dev_info.ublksrv_pid),
		Flags:         uint64(dev.dev_info.flags),
		UblksrvFlags:  uint64(dev.dev_info.ublksrv_flags),
		OwnerUID:      uint32(dev.dev_info.owner_uid),
		OwnerGID:      uint32(dev.dev_info.owner_gid),
		Size:          int64(params.basic.dev_sectors) << 9,
		BlockSize:     1 << params.basic.logical_bs_shift,
	}

	return info, nil
}
