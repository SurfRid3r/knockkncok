
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func initCmd() {
	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		fmt.Println("Error generating key:", err)
		os.Exit(1)
	}

	fmt.Println("key = \"" + base64.StdEncoding.EncodeToString(key) + "\"")
}
