// client.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func dialWithRetry(addr string) net.Conn {
	for {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			return conn
		}
		log.Printf("connect failed: %v; retrying in 1s...", err)
		time.Sleep(time.Second)
	}
}

func main() {
	addr := "127.0.0.1:4000"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	fmt.Printf("Connecting to %s...\n", addr)

	conn := dialWithRetry(addr)
	defer conn.Close()
	fmt.Println("Connected. Type commands: PUT <key> <value>, GET <key>, DELETE <key>, LIST, QUIT")

	// Reader for server responses
	serverReader := bufio.NewReader(conn)
	// Reader for stdin (user)
	stdinReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		text, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Println("stdin error:", err)
			return
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		// send command to server
		_, err = fmt.Fprintln(conn, text)
		if err != nil {
			// connection may be dead; try to reconnect
			log.Println("write error, attempting reconnect:", err)
			conn.Close()
			conn = dialWithRetry(addr)
			serverReader = bufio.NewReader(conn)
			_, _ = fmt.Fprintln(conn, text) // try send again
		}

		// Read server response(s)
		// LIST returns multiple lines terminated by "END" (or "EMPTY")
		tokens := strings.Fields(text)
		cmd := strings.ToUpper(tokens[0])

		for {
			line, err := serverReader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					log.Println("server closed connection; reconnecting...")
					conn = dialWithRetry(addr)
					serverReader = bufio.NewReader(conn)
					break
				}
				log.Println("read error:", err)
				break
			}
			line = strings.TrimRight(line, "\r\n")
			fmt.Println(line)

			// stop reading when single-line responses are received,
			// or when LIST's terminator "END" or "EMPTY" is seen.
			if cmd == "LIST" {
				if line == "END" || line == "EMPTY" {
					break
				}
				// continue reading until END
				continue
			} else {
				// single line response for other commands
				break
			}
		}

		if strings.ToUpper(text) == "QUIT" {
			fmt.Println("Disconnected.")
			return
		}
	}
}
