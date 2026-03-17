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

		proCmd, err := buildCommand([]string{"GET", args[0]})
		if err != nil {
			fmt.Println("Build command error:", err)
			return
		}

		err = protocol.WriteCommand(conn, proCmd)
		if err != nil {
			fmt.Println("Write command error:", err)
			return
		}

		resp, err := protocol.ReadResponse(conn)
		if err != nil {
			fmt.Println("Read response error:", err)
			return
		}

		printResponse(resp)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	// rootCmd.AddCommand(setCmd)
	// rootCmd.AddCommand(delCmd)
	// rootCmd.AddCommand(incrCmd)
	// rootCmd.AddCommand(ttlCmd)
	// rootCmd.AddCommand(keysCmd)
	// rootCmd.AddCommand(expireCmd)
	// rootCmd.AddCommand(pingCmd)
}
