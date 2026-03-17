package main

import (
	"fmt"
	"strings"

	"github.com/ARCoder181105/kvstore/internal/protocol"
)

func main() {
	Execute()
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
		count := 0
		for _, line := range lines {
			if line != "" {
				count++
				fmt.Printf("%d) %s\n", count, line)
			}
		}
		if count == 0 {
			fmt.Println("(empty)")
		}
	case protocol.StatusError:
		fmt.Println("ERROR:", string(resp.Payload))
	default:
		fmt.Printf("unknown response status: 0x%02x\n", resp.Status)
	}
}
