package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kvcli",
	Short: "A CLI client for kvstore",
	Run: func(cmd *cobra.Command, args []string) {
		// when no subcommand is given, start the REPL
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		startREPL(host, port)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("host", "localhost", "server host")
	rootCmd.PersistentFlags().Int("port", 6379, "server port")
}
