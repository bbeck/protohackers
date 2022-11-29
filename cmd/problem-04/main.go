package main

import (
	"fmt"
	"github.com/bbeck/protohackers/internal"
	"net"
	"strings"
)

func main() {
	db := map[string]string{
		"version": "alpha",
	}

	internal.RunUDPServer(func(_ net.Addr, bs []byte, send func([]byte)) {
		s := string(bs)

		if key, value, found := strings.Cut(s, "="); found {
			if key != "version" {
				db[key] = value
			}
			return
		}

		send([]byte(fmt.Sprintf("%s=%s", s, db[s])))
	})
}
