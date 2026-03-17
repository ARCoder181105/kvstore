package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/spf13/cobra"
)

func connect(host string, port int) (net.Conn, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	return net.Dial("tcp", addr)
}

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a value by key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Root().PersistentFlags().GetString("host")
		port, _ := cmd.Root().PersistentFlags().GetInt("port")

		conn, err := connect(host, port)
		if err != nil {
			fmt.Println("Connection error:", err)
			return
		}
		defer conn.Close()

		if err := runCommand(conn, []string{"GET", args[0]}); err != nil {
			fmt.Println("Error:", err)
		}
	},
}

var setCmd = &cobra.Command{
	Use:   "set <key> <value> [ttl]",
	Short: "Set a value by key",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Root().PersistentFlags().GetString("host")
		port, _ := cmd.Root().PersistentFlags().GetInt("port")

		conn, err := connect(host, port)
		if err != nil {
			fmt.Println("Connection error:", err)
			return
		}
		defer conn.Close()

		tokens := []string{"SET", args[0], args[1]}
		if len(args) == 3 {
			tokens = append(tokens, args[2])
		}

		if err := runCommand(conn, tokens); err != nil {
			fmt.Println("Error:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(setCmd)
	// rootCmd.AddCommand(delCmd)
	// rootCmd.AddCommand(incrCmd)
	// rootCmd.AddCommand(ttlCmd)
	// rootCmd.AddCommand(keysCmd)
	// rootCmd.AddCommand(expireCmd)
	// rootCmd.AddCommand(pingCmd)
}

func runCommand(conn net.Conn, tokens []string) error {
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
