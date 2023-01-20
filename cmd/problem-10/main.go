package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/bbeck/protohackers/internal"
	"io"
	"net"
	"strconv"
	"strings"
)

func main() {
	fs := NewFilesystem()

	internal.RunTCPServer(func(conn net.Conn) {
		defer func(c io.Closer) { _ = c.Close() }(conn)

		client := &Client{
			reader: bufio.NewReader(conn),
			writer: conn,
		}

		for {
			client.Send("READY")

			cmd, args := client.ReadCommand()
			if cmd == "" { // This is probably an EOF
				break
			}

			switch cmd {
			case "GET":
				if len(args) == 0 || len(args) > 2 {
					client.Send("ERR usage: GET file [revision]")
					continue
				}
				for len(args) < 2 {
					args = append(args, "")
				}

				filename := args[0]
				if !IsLegalDirectoryOrFilename(filename) {
					client.Send("ERR illegal file name")
					continue
				}

				revision, err := ParseRevision(args[1])
				bs := fs.Get(filename, revision)
				if err != nil || bs == nil {
					client.Send("ERR no such revision")
					continue
				}

				client.Send("OK %d", len(bs))
				client.SendBytes(bs)

			case "HELP":
				client.Send("OK usage: HELP|GET|PUT|LIST")

			case "LIST":
				if len(args) != 1 {
					client.Send("ERR usage: LIST dir")
					continue
				}

				dir := args[0]
				if !IsLegalDirectoryOrFilename(dir) {
					client.Send("ERR illegal dir name")
					continue
				}

				entries := fs.ListDir(dir)
				client.Send("OK %d", len(entries))
				for _, entry := range entries {
					if len(entry.Revisions) > 0 {
						// This is a file
						client.Send("%s r%d", entry.Name, len(entry.Revisions))
						continue
					}

					// This is a directory
					client.Send("%s/ DIR", entry.Name)
				}

			case "PUT":
				if len(args) != 2 {
					client.Send("ERR usage: PUT file length newline data")
					continue
				}

				filename := args[0]
				if !IsLegalDirectoryOrFilename(filename) {
					client.Send("ERR illegal file name")
					continue
				}

				length, err := ParseLength(args[1])
				if err != nil {
					client.Send("ERR %s", err.Error())
					continue
				}

				bs := client.ReadBytes(length)
				if !IsLegalPayload(bs) {
					client.Send("ERR illegal payload")
					continue
				}

				revision := fs.Put(filename, bs)
				client.Send("OK r%d", revision)

			default:
				client.Send("ERR illegal method: %s", cmd)
			}
		}
	})
}

func ParseLength(s string) (int, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0, nil
	}

	if n == 0 {
		return 0, errors.New("zero length")
	}

	return int(n), nil
}

func ParseInt(s string) (int, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	return int(n), err
}

func ParseRevision(s string) (int, error) {
	if len(s) == 0 {
		return -1, nil
	}

	if s[0] == 'r' {
		s = s[1:]
	}

	return ParseInt(s)
}

func IsLegalDirectoryOrFilename(name string) bool {
	if !strings.HasPrefix(name, "/") {
		return false
	}

	if strings.ContainsAny(name, "!@#$%^&*()=+[{]}|'\";:<>`~?") {
		return false
	}

	if strings.Contains(name, "//") {
		return false
	}

	return true
}

func IsLegalPayload(bs []byte) bool {
	return bytes.IndexFunc(bs, func(r rune) bool {
		if r == 0x09 || r == 0x0A || r == 0x0D {
			return false
		}

		return r < 0x20 || r >= 0x7F
	}) == -1
}

type Client struct {
	reader *bufio.Reader
	writer io.Writer
}

func (c *Client) Send(s string, args ...any) {
	line := fmt.Sprintf(s, args...)
	if !strings.HasSuffix(line, "\n") {
		line = line + "\n"
	}

	_, _ = io.WriteString(c.writer, line)
}

func (c *Client) SendBytes(bs []byte) {
	_, _ = c.writer.Write(bs)
}

func (c *Client) ReadCommand() (string, []string) {
	line, err := c.reader.ReadString('\n')
	fields := strings.Fields(line)
	if err != nil || len(fields) == 0 {
		return "", nil
	}

	return strings.ToUpper(fields[0]), fields[1:]
}

func (c *Client) ReadBytes(n int) []byte {
	bs := make([]byte, n)
	_, _ = io.ReadFull(c.reader, bs)
	return bs
}
