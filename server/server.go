
// server.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type KVStore struct {
	mu sync.RWMutex
	m  map[string]string
}

func NewKVStore() *KVStore {
	return &KVStore{
		m: make(map[string]string),
	}
}

func (s *KVStore) Put(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

func (s *KVStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.m[key]
	return val, ok
}

func (s *KVStore) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[key]; ok {
		delete(s.m, key)
		return true
	}
	return false
}

func (s *KVStore) List() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// return a copy to avoid callers accidentally holding references
	cp := make(map[string]string, len(s.m))
	for k, v := range s.m {
		cp[k] = v
	}
	return cp
}

func handleConnection(conn net.Conn, store *KVStore) {
	defer conn.Close()
	addr := conn.RemoteAddr().String()
	log.Printf("client connected: %s\n", addr)
	r := bufio.NewReader(conn)

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Printf("client disconnected: %s\n", addr)
			} else {
				log.Printf("read error from %s: %v\n", addr, err)
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse command
		parts := strings.Fields(line)
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "PUT":
			// PUT <key> <value...>
			if len(parts) < 2 {
				fmt.Fprintln(conn, "ERR usage: PUT <key> <value>")
				continue
			}
			key := parts[1]
			// value is the remainder of the line after the key
			idx := strings.Index(line, key)
			value := ""
			if idx >= 0 {
				// skip key and a space
				afterKey := line[idx+len(key):]
				value = strings.TrimSpace(afterKey)
			}
			store.Put(key, value)
			fmt.Fprintln(conn, "OK")

		case "GET":
			// GET <key>
			if len(parts) != 2 {
				fmt.Fprintln(conn, "ERR usage: GET <key>")
				continue
			}
			key := parts[1]
			if val, ok := store.Get(key); ok {
				fmt.Fprintf(conn, "VALUE %s\n", val)
			} else {
				fmt.Fprintln(conn, "ERR key not found")
			}

		case "DELETE":
			// DELETE <key>
			if len(parts) != 2 {
				fmt.Fprintln(conn, "ERR usage: DELETE <key>")
				continue
			}
			key := parts[1]
			if ok := store.Delete(key); ok {
				fmt.Fprintln(conn, "OK")
			} else {
				fmt.Fprintln(conn, "ERR key not found")
			}

		case "LIST":
			// LIST
			entries := store.List()
			if len(entries) == 0 {
				fmt.Fprintln(conn, "EMPTY")
				continue
			}
			// send each key=value line, then a terminator line "END"
			for k, v := range entries {
				fmt.Fprintf(conn, "%s=%s\n", k, v)
			}
			fmt.Fprintln(conn, "END")

		case "QUIT":
			fmt.Fprintln(conn, "BYE")
			log.Printf("client requested quit: %s\n", addr)
			return

		default:
			fmt.Fprintln(conn, "ERR unknown command. Supported: PUT, GET, DELETE, LIST, QUIT")
		}
	}
}

func main() {
	addr := ":4000" // listen on port 4000
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v\n", addr, err)
	}
	defer listener.Close()
	log.Printf("kv server listening on %s\n", addr)

	store := NewKVStore()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept error: %v\n", err)
			continue
		}
		go handleConnection(conn, store) // goroutine per client
	}
}
