package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

const (
	CONN_PORT = ":9090"
	CONN_TYPE = "tcp"
)

func readFromServer(conn net.Conn) {
	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected from the server.")
			return
		}
		fmt.Print("Server: ", message)
	}
}

func writeToServer(conn net.Conn) {
	consoleReader := bufio.NewReader(os.Stdin)
	for {
		input, _ := consoleReader.ReadString('\n')

		trimmedInput := strings.TrimSpace(input)
		if trimmedInput == "/quit" {
			fmt.Println("Disconnecting from server...")
			conn.Write([]byte("/quit\n"))
			conn.Close()
			os.Exit(0)
		}

		conn.Write([]byte(input))
	}
}

func main() {
	conn, err := net.Dial(CONN_TYPE, CONN_PORT)
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}

	fmt.Print("Enter your nickname: ")
	nicknameReader := bufio.NewReader(os.Stdin)
	nickname, _ := nicknameReader.ReadString('\n')
	nickname = strings.TrimSpace(nickname)
	conn.Write([]byte("/nickname " + nickname + "\n"))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		readFromServer(conn)
	}()

	writeToServer(conn)

	wg.Wait()
}
