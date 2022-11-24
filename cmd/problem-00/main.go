package main

import (
	"github.com/bbeck/protohackers/internal"
	"io"
	"net"
)

func main() {
	internal.RunWithTunnel(func(conn net.Conn) {
		defer conn.Close()
		io.Copy(conn, conn)
	})
}
