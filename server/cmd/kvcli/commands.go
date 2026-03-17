package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/spf13/cobra"
)

func connect(host string, port int) (net.Conn, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	return net.Dial("tcp", addr)
}

// hostPort extracts --host and --port from the root persistent flags.
func hostPort(cmd *cobra.Command) (string, int) {
	host, _ := cmd.Root().PersistentFlags().GetString("host")
	port, _ := cmd.Root().PersistentFlags().GetInt("port")
	return host, port
}

func sendAndPrint(host string, port int, tokens []string) {
	conn, err := connect(host, port)
	if err != nil {
		fmt.Println("connection error:", err)
		return
	}
	defer conn.Close()

	if err := runCommand(conn, tokens); err != nil {
		fmt.Println("error:", err)
	}
}

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get the value of a key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host, port := hostPort(cmd)
		sendAndPrint(host, port, []string{"GET", args[0]})
	},
}

// actually parse the optional TTL argument (previously it was appended
// to the token slice but buildCommand never read it, so TTL was always 0).
var setCmd = &cobra.Command{
	Use:   "set <key> <value> [ttl_seconds]",
	Short: "Set a key to a value with an optional TTL in seconds",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		host, port := hostPort(cmd)
		tokens := append([]string{"SET"}, args...) // SET key value [ttl]
		sendAndPrint(host, port, tokens)
	},
}

var delCmd = &cobra.Command{
	Use:   "del <key>",
	Short: "Delete a key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host, port := hostPort(cmd)
		sendAndPrint(host, port, []string{"DEL", args[0]})
	},
}

var incrCmd = &cobra.Command{
	Use:   "incr <key>",
	Short: "Increment the integer value of a key by 1",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host, port := hostPort(cmd)
		sendAndPrint(host, port, []string{"INCR", args[0]})
	},
}

var ttlCmd = &cobra.Command{
	Use:   "ttl <key>",
	Short: "Get the remaining TTL of a key in seconds (-1 = no expiry, -2 = not found)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host, port := hostPort(cmd)
		sendAndPrint(host, port, []string{"TTL", args[0]})
	},
}

var keysCmd = &cobra.Command{
	Use:   "keys <pattern>",
	Short: "List all keys matching a glob pattern (e.g. \"user:*\")",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host, port := hostPort(cmd)
		sendAndPrint(host, port, []string{"KEYS", args[0]})
	},
}

var expireCmd = &cobra.Command{
	Use:   "expire <key> <seconds>",
	Short: "Set a TTL on an existing key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		host, port := hostPort(cmd)
		sendAndPrint(host, port, []string{"EXPIRE", args[0], args[1]})
	},
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping the server",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := hostPort(cmd)
		sendAndPrint(host, port, []string{"PING"})
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(delCmd)
	rootCmd.AddCommand(incrCmd)
	rootCmd.AddCommand(ttlCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(expireCmd)
	rootCmd.AddCommand(pingCmd)
}

func runCommand(conn net.Conn, tokens []string) error {
	cmd, err := buildCommand(tokens)
	if err != nil {
		return err
	}
	if err := protocol.WriteCommand(conn, cmd); err != nil {
		return err
	}
	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		return err
	}
	printResponse(resp)
	return nil
}

// buildCommand converts a slice of string tokens into a protocol.Command.
// SET now correctly parses the optional third token as a TTL in seconds
// and converts it to nanoseconds for the wire protocol.
func buildCommand(args []string) (*protocol.Command, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command given")
	}

	switch args[0] {
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
		cmd := &protocol.Command{
			ID:    protocol.CmdSet,
			Key:   args[1],
			Value: []byte(args[2]),
		}
		// parse optional TTL argument
		if len(args) >= 4 {
			secs, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil || secs <= 0 {
				return nil, fmt.Errorf("TTL must be a positive integer, got %q", args[3])
			}
			cmd.TTL = secs * int64(time.Second)
		}
		return cmd, nil

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
		secs, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil || secs <= 0 {
			return nil, fmt.Errorf("TTL must be a positive integer, got %q", args[2])
		}
		return &protocol.Command{
			ID:  protocol.CmdExpire,
			Key: args[1],
			TTL: secs * int64(time.Second),
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", args[0])
	}
}
