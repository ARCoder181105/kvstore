package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kvcli",
	Short: "A CLI client for kvstore",
	Long:  "kvcli — interactive REPL and single-command client for the kvstore TCP server.",
	Run: func(cmd *cobra.Command, args []string) {
		// No subcommand given → start the REPL
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		startREPL(host, port)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("host", "localhost", "server host")
	rootCmd.PersistentFlags().Int("port", 6379, "server port")
}
