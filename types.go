package GoUblksrv

type IOHandler interface {
	ReadAt(p []byte, off int64) (int, error)
	WriteAt(p []byte, off int64) (int, error)
	UnmapAt(length uint32, off int64) (int, error)
}

const (
	TypeRead = iota
	TypeWrite
	TypeResponse
	TypeError
	TypeEOF
	TypeClose
	TypePing
	TypeUnmap
)

const (
	StateDead = iota
	StateLive
	StateQuiesced
	StateFailIo
)

const (
	MagicVersion = uint16(0x1b01)
	Blocks       = 512
)

type Request struct {
	Complete     chan struct{}
	MagicVersion uint16
	Seq          uint32
	Type         uint32
	Offset       int64
	Size         uint32
	RData        []byte
	WData        []byte
	Result       int
}

type DeviceInfo struct {
	ID            int
	NrHwQueues    int
	QueueDepth    int
	State         int
	MaxIoBufBytes uint32
	UblksrvPid    int
	Flags         uint64
	UblksrvFlags  uint64
	OwnerUID      uint32
	OwnerGID      uint32
	Size          int64
	BlockSize     int
}
