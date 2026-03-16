package protocol

const (
	CmdSet    byte = 0x01 // Set key to value
	CmdGet    byte = 0x02 // Get value by key
	CmdDel    byte = 0x03 // Delete key
	CmdExpire byte = 0x04 // Set key expiration
	CmdTTL    byte = 0x05 // Get key TTL
	CmdKeys   byte = 0x06 // List all keys
	CmdIncr   byte = 0x07 // Increment integer value
	CmdMSet   byte = 0x08 // Set multiple keys
	CmdMGet   byte = 0x09 // Get multiple keys
	CmdPing   byte = 0x0A // Ping server
)

const (
	StatusOK    byte = 0x00 // Command succeeded, no payload needed (SET, DEL, EXPIRE)
	StatusError byte = 0x01 // Error — payload contains error message string
	StatusValue byte = 0x02 // Success — payload contains the value bytes (GET)
	StatusNull  byte = 0x03 // Key not found (GET on missing key)
	StatusInt   byte = 0x04 // Success — payload contains int64 as string (TTL, INCR)
	StatusArray byte = 0x05 // Success — payload contains newline-separated strings (KEYS)
)

type Command struct {
    ID     byte
    Key    string
    Value  []byte
    TTL    int64
    // Keys   []string  // for MGet
    // Values [][]byte  // for MSet
}

type Response struct {
	Status  byte
	Payload []byte
}
