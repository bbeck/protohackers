package internal

import (
	"context"
	"fmt"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"log"
	"net"
)

func RunWithTunnel(fn func(conn net.Conn)) {
	tunnel, err := ngrok.StartTunnel(
		context.Background(),
		config.TCPEndpoint(),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		log.Fatalf("error creating ngrok tunnel: %v", err)
	}
	defer tunnel.Close()

	host, port, _ := net.SplitHostPort(tunnel.Addr().String())
	ips, err := net.LookupIP(host)
	if err != nil {
		log.Fatalf("error looking up ip for hostname %s: %v", host, err)
	}

	var ip string
	for i := range ips {
		if ipv4 := ips[i].To4(); ipv4 != nil {
			ip = ipv4.String()
			break
		}
	}
	fmt.Printf("Listening on: %s %s\n", ip, port)

	for {
		conn, err := tunnel.Accept()
		if err != nil {
			log.Printf("error accepting connection from %v: %v", conn.RemoteAddr(), err)
			continue
		}

		go fn(conn)
	}
}
