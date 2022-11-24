package main

import (
	"bufio"
	"fmt"
	"github.com/bbeck/protohackers/internal"
	"io"
	"net"
	"sync"
)

func main() {
	var room Room

	internal.RunTCPServer(func(conn net.Conn) {
		defer conn.Close()

		scanner := bufio.NewScanner(conn)

		// Read name
		io.WriteString(conn, "Name:\n")
		if !scanner.Scan() {
			return
		}
		name := scanner.Text()
		if !IsValidName(name) {
			return
		}

		// Join the room
		room.Join(name, conn)
		defer room.Part(name)

		// Now that the user is connected keep sending their messages until
		// they disconnect
		for scanner.Scan() {
			room.Send(name, scanner.Text())
		}
	})
}

func IsValidName(name string) bool {
	for _, r := range name {
		isValid := ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9')
		if !isValid {
			return false
		}
	}
	return len(name) > 0
}

type Room struct {
	sync.Mutex
	Members map[string]net.Conn
}

func (r *Room) Join(name string, conn net.Conn) {
	r.Lock()
	defer r.Unlock()

	r.send(name, fmt.Sprintf("* %s joined\n", name))
	io.WriteString(conn, fmt.Sprintf("* members: %v\n", r.Members))

	if r.Members == nil {
		r.Members = make(map[string]net.Conn)
	}
	r.Members[name] = conn
}

func (r *Room) Part(name string) {
	r.Lock()
	defer r.Unlock()

	delete(r.Members, name)
	r.send(name, fmt.Sprintf("* %s left\n", name))
}

func (r *Room) Send(name, message string) {
	r.Lock()
	defer r.Unlock()

	r.send(name, fmt.Sprintf("[%s] %s\n", name, message))
}

func (r *Room) send(name, msg string) {
	for m, conn := range r.Members {
		if m == name {
			continue
		}

		io.WriteString(conn, msg)
	}
}
