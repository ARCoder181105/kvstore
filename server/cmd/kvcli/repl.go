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
		fmt.Println("TCP connection error:", err)
		return
	}
	defer conn.Close()

	rl, err := readline.NewEx(&readline.Config{
		Prompt: "kvstore> ",
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

		proCmd, err := buildCommand(tokens)
		if err != nil {
			fmt.Println("Failed to Build Command in REPL: ", err)
			continue
		}

		err = protocol.WriteCommand(conn, proCmd)
		if err != nil {
			fmt.Println("Failed to Write Command in REPL: ", err)
			return
		}

		resp, err := protocol.ReadResponse(conn)
		if err != nil {
			fmt.Print("Error in reading the response in REPL: ", err)
			return
		}

		printResponse(resp)
	}
}
