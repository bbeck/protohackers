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

func RunUDPServer(handler func(bs []byte) []byte) {
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:40000")
	if err != nil {
		log.Fatalf("error resolving UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("error listening for UDP connections: %v", err)
	}
	defer conn.Close()

	for {
		buffer := make([]byte, 1024*1024)
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return
		}

		response := handler(buffer[:n])
		if response != nil {
			_, err := conn.WriteTo(response, addr)
			if err != nil {
				return
			}
		}
	}
}
