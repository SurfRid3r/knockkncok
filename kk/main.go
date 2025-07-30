
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: kk <init|send>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		initCmd()
	case "send":
		sendFlags := flag.NewFlagSet("send", flag.ExitOnError)
		serverIP := sendFlags.String("s", "", "Server IP address")
		key := sendFlags.String("k", "", "Master key (base64)")
		sendFlags.Parse(os.Args[2:])

		if *serverIP == "" || *key == "" {
			fmt.Println("Usage: kk send -s <server_ip> -k <key>")
			os.Exit(1)
		}
		sendCmd(*serverIP, *key)
	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}
