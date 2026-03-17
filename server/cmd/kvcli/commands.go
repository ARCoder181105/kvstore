package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/spf13/cobra"
)

// connect handles safely joining the host and port, supporting both IPv4 and IPv6.
func connect(host string, port int) (net.Conn, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	return net.Dial("tcp", addr)
}

// execute is a helper function that abstracts the connection setup, execution, and teardown.
func execute(cmd *cobra.Command, tokens []string) {
	host, _ := cmd.Root().PersistentFlags().GetString("host")
	port, _ := cmd.Root().PersistentFlags().GetInt("port")

	conn, err := connect(host, port)
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	defer conn.Close()

	if err := runCommandREPL(conn, tokens); err != nil {
		fmt.Println("Error:", err)
	}
}

// runCommand handles the protocol-specific building, writing, reading, and printing.
func runCommandREPL(conn net.Conn, tokens []string) error {
	proCmd, err := buildCommand(tokens)
	if err != nil {
		return err
	}

	err = protocol.WriteCommand(conn, proCmd)
	if err != nil {
		return err
	}

	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		return err
	}

	printResponse(resp)
	return nil
}

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a value by key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		execute(cmd, []string{"GET", args[0]})
	},
}

var setCmd = &cobra.Command{
	Use:   "set <key> <value> [ttl]",
	Short: "Set a value by key",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		tokens := []string{"SET", args[0], args[1]}
		if len(args) == 3 {
			tokens = append(tokens, args[2])
		}
		execute(cmd, tokens)
	},
}

var delCmd = &cobra.Command{
	Use:   "del <key>",
	Short: "Delete key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		execute(cmd, []string{"DEL", args[0]})
	},
}

var incrCmd = &cobra.Command{
	Use:   "incr <key>",
	Short: "Increment integer value by key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		execute(cmd, []string{"INCR", args[0]})
	},
}

var ttlCmd = &cobra.Command{
	Use:   "ttl <key>",
	Short: "Get TTL for a key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		execute(cmd, []string{"TTL", args[0]})
	},
}

var keysCmd = &cobra.Command{
	Use:   "keys <pattern>",
	Short: "List keys matching pattern",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		execute(cmd, []string{"KEYS", args[0]})
	},
}

var expireCmd = &cobra.Command{
	Use:   "expire <key> <ttl>",
	Short: "Set TTL for a key",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		execute(cmd, []string{"EXPIRE", args[0], args[1]})
	},
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping the server",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		execute(cmd, []string{"PING"})
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
