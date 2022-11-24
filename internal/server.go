package internal

import (
	"log"
	"net"
)

func RunTCPServer(handler func(conn net.Conn)) {
	addr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:40000")
	if err != nil {
		log.Fatalf("error resolving TCP address: %v", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("error listening for TCP connections: %v", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("error accepting connection from %v: %v", conn.RemoteAddr(), err)
			continue
		}

		go handler(conn)
	}
}
