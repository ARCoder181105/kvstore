package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
)

func startREPL(host string, port int) {
	conn, err := connect(host, port)
	if err != nil {
		fmt.Println("TCP connection error:", err)
		return
	}
	defer conn.Close()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:      "kvstore> ",
		HistoryFile: "/tmp/kvstore_history",
	})
	if err != nil {
		fmt.Println("Cannot Initialize Repel error:", err)
		return
	}
	defer rl.Close()

	for {

		line, err := rl.Readline()
		if err == io.EOF {
			fmt.Println("bye")
			return
		}
		if err == readline.ErrInterrupt {
			fmt.Println("bye")
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		tokens := strings.Fields(line)
		switch strings.ToUpper(tokens[0]) {
		case "EXIT", "QUIT":
			fmt.Println("bye")
			return
		case "HELP":
			fmt.Println("Commands: SET, GET, DEL, INCR, TTL, KEYS, EXPIRE, PING")
			continue
		}

		if err := runCommandREPL(conn, tokens); err != nil {
			fmt.Println("Error:", err)
			return // connection broken
		}
	}
}
