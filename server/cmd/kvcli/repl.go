package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/chzyer/readline"
)

func startREPL(host string, port int) {
	conn, err := connect(host, port)
	if err != nil {
		fmt.Println("connection error:", err)
		return
	}
	defer conn.Close()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "kvstore> ",
		HistoryFile:     "/tmp/kvcli_history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Println("readline init error:", err)
		return
	}
	defer rl.Close()

	fmt.Println("kvstore REPL — type HELP for commands, EXIT to quit")

	for {
		line, err := rl.Readline()
		if err == io.EOF || err == readline.ErrInterrupt {
			fmt.Println("bye")
			return
		}
		if err != nil {
			fmt.Println("read error:", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		tokens := strings.Fields(line)
		// Normalise command name to uppercase so "get key" and "GET key" both work
		tokens[0] = strings.ToUpper(tokens[0])

		switch tokens[0] {
		case "EXIT", "QUIT":
			fmt.Println("bye")
			return

		case "HELP":
			fmt.Println("Commands: SET key value [ttl_seconds]")
			fmt.Println("          GET key")
			fmt.Println("          DEL key")
			fmt.Println("          INCR key")
			fmt.Println("          TTL key")
			fmt.Println("          KEYS pattern")
			fmt.Println("          EXPIRE key seconds")
			fmt.Println("          PING")
			fmt.Println("          EXIT / QUIT")
			continue
		}

		cmd, err := buildCommand(tokens)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}

		if err := protocol.WriteCommand(conn, cmd); err != nil {
			fmt.Println("write error:", err)
			return
		}

		resp, err := protocol.ReadResponse(conn)
		if err != nil {
			fmt.Println("read error:", err)
			return
		}

		printResponse(resp)
	}
}
