package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s IP PORT USER PASS\n", os.Args[0])
}

func main() {
	if len(os.Args) != 5 {
		usage()
		return
	}
	config := &ssh.ClientConfig {
		User: os.Args[3],
		Auth: []ssh.AuthMethod {
			ssh.Password(os.Args[4]),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn := fmt.Sprintf("%s:%s", os.Args[1], os.Args[2])
	_, err := ssh.Dial("tcp", conn, config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Success")
}
