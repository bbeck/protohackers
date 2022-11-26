package main

import (
	"bufio"
	"github.com/bbeck/protohackers/internal"
	"io"
	"log"
	"net"
	"regexp"
	"strings"
)

const (
	UpstreamAddress = "chat.protohackers.com:16963"
	TonyAddress     = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"
)

func main() {
	internal.RunTCPServer(func(conn net.Conn) {
		defer conn.Close()

		upstream, err := net.Dial("tcp", UpstreamAddress)
		if err != nil {
			log.Fatalf("error dialing TCP address: %v", err)
		}
		defer upstream.Close()

		toConn, toUpstream := make(chan string, 10), make(chan string, 10)
		go Transform(bufio.NewReader(conn), toUpstream)
		go Transform(bufio.NewReader(upstream), toConn)

		for {
			select {
			case msg, ok := <-toConn:
				if !ok {
					return
				}
				io.WriteString(conn, msg)

			case msg, ok := <-toUpstream:
				if !ok {
					return
				}
				io.WriteString(upstream, msg)
			}
		}
	})
}

var Regex = regexp.MustCompile(`^7[0-9a-zA-Z]{25,34}$`)

func Transform(in *bufio.Reader, ch chan string) {
	defer close(ch)

	for {
		msg, err := in.ReadString('\n')
		if err != nil {
			return
		}

		for _, word := range strings.Fields(msg) {
			if Regex.MatchString(word) {
				msg = strings.ReplaceAll(msg, word, TonyAddress)
			}
		}

		ch <- msg
	}
}
