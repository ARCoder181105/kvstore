package protocol

const (
	CmdSet    byte = 0x01
	CmdGet    byte = 0x02
	CmdDel    byte = 0x03
	CmdExpire byte = 0x04
	CmdTTL    byte = 0x05
	CmdKeys   byte = 0x06
	CmdIncr   byte = 0x07
	CmdMSet   byte = 0x08
	CmdMGet   byte = 0x09
	CmdPing   byte = 0x0A
)

const (
	StatusOK    byte = 0x00
	StatusError byte = 0x01
	StatusValue byte = 0x02
	StatusNull  byte = 0x03
	StatusInt   byte = 0x04
	StatusArray byte = 0x05
)

type Command struct {
	ID    byte
	Key   string
	Value []byte
	TTL   int64 // nanoseconds
}

type Response struct {
	Status  byte
	Payload []byte
}
