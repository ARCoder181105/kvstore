package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ARCoder181105/kvstore/internal/protocol"
)

func main() {
	// check args
	if len(os.Args) < 2 {
		fmt.Println("usage: kvcli <command> [args]")
		fmt.Println("commands: SET, GET, DEL, PING, KEYS, TTL, INCR, EXPIRE")
		os.Exit(1)
	}

	// connect to server
	conn, err := net.Dial("tcp", ":6379")
	if err != nil {
		fmt.Println("failed to connect to server:", err)
		os.Exit(1)
	}
	defer conn.Close()

	// build command from args
	cmd, err := buildCommand(os.Args[1:])
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	// send command
	err = protocol.WriteCommand(conn, cmd)
	if err != nil {
		fmt.Println("failed to send command:", err)
		os.Exit(1)
	}

	// read response
	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		fmt.Println("failed to read response:", err)
		os.Exit(1)
	}

	// print response
	printResponse(resp)
}

func buildCommand(args []string) (*protocol.Command, error) {
	cmdName := strings.ToUpper(args[0])

	switch cmdName {
	case "PING":
		return &protocol.Command{ID: protocol.CmdPing}, nil

	case "GET":
		if len(args) < 2 {
			return nil, fmt.Errorf("GET requires a key")
		}
		return &protocol.Command{ID: protocol.CmdGet, Key: args[1]}, nil

	case "SET":
		if len(args) < 3 {
			return nil, fmt.Errorf("SET requires a key and value")
		}
		return &protocol.Command{
			ID:    protocol.CmdSet,
			Key:   args[1],
			Value: []byte(args[2]),
		}, nil

	case "DEL":
		if len(args) < 2 {
			return nil, fmt.Errorf("DEL requires a key")
		}
		return &protocol.Command{ID: protocol.CmdDel, Key: args[1]}, nil

	case "INCR":
		if len(args) < 2 {
			return nil, fmt.Errorf("INCR requires a key")
		}
		return &protocol.Command{ID: protocol.CmdIncr, Key: args[1]}, nil

	case "TTL":
		if len(args) < 2 {
			return nil, fmt.Errorf("TTL requires a key")
		}
		return &protocol.Command{ID: protocol.CmdTTL, Key: args[1]}, nil

	case "KEYS":
		if len(args) < 2 {
			return nil, fmt.Errorf("KEYS requires a pattern")
		}
		return &protocol.Command{ID: protocol.CmdKeys, Key: args[1]}, nil

	case "EXPIRE":
		if len(args) < 3 {
			return nil, fmt.Errorf("EXPIRE requires a key and seconds")
		}
		seconds, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid TTL: %v", err)
		}
		return &protocol.Command{
			ID:  protocol.CmdExpire,
			Key: args[1],
			TTL: seconds * int64(time.Second),
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

func printResponse(resp *protocol.Response) {
	switch resp.Status {
	case protocol.StatusOK:
		fmt.Println("OK")
	case protocol.StatusValue:
		fmt.Println(string(resp.Payload))
	case protocol.StatusNull:
		fmt.Println("(nil)")
	case protocol.StatusInt:
		fmt.Println(string(resp.Payload))
	case protocol.StatusArray:
		lines := strings.Split(string(resp.Payload), "\n")
		for i, line := range lines {
			if line != "" {
				fmt.Printf("%d) %s\n", i+1, line)
			}
		}
	case protocol.StatusError:
		fmt.Println("ERROR:", string(resp.Payload))
	default:
		fmt.Println("unknown response status:", resp.Status)
	}
}
