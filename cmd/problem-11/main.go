package main

import (
	"github.com/bbeck/protohackers/internal"
	"io"
	"net"
)

func main() {
	authorities := NewAuthorities()

	internal.RunTCPServer(func(conn net.Conn) {
		defer func(c io.Closer) { _ = c.Close() }(conn)

		if err := WriteMessage(conn, HelloMessage{}); err != nil {
			return
		}

		if _, err := ReadMessage[HelloMessage](conn); err != nil {
			_ = WriteMessage(conn, Error{Message: err.Error()})
			return
		}

		for {
			m, err := ReadMessage[SiteVisit](conn)
			if err != nil {
				_ = WriteMessage(conn, Error{Message: err.Error()})
				return
			}

			a, err := authorities.GetAuthority(m.Site)
			if err != nil {
				return
			}

			a.HandleSiteVisit(m)
		}
	})
}
