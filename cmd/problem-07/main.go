package main

import (
	"errors"
	"fmt"
	"github.com/bbeck/protohackers/internal"
	"net"
	"strconv"
)

func main() {
	sessions := NewSessions()

	internal.RunUDPServer(func(_ net.Addr, bs []byte, send func([]byte)) {
		packet, err := ParsePacket(bs)
		if err != nil {
			return
		}

		// If the session is not open: send /close/SESSION/ and stop, unless this
		// is a "connect" message.
		session := sessions.Get(packet.Session)
		if session == nil && !packet.IsConnect {
			send([]byte(fmt.Sprintf("/close/%d/", packet.Session)))
			return
		}

		if packet.IsConnect {
			// The spec says we can assume that the peer for any given session is at a
			// fixed ip/port.  Because of that we can cache the send method and use it
			// later.
			if session == nil {
				session = NewSession(packet.Session, send)
				sessions.Put(packet.Session, session)
			}

			session.HandleConnect()
		}

		if packet.IsData {
			session.HandleData(packet.Pos, packet.Data)
		}

		if packet.IsAck {
			session.HandleAck(packet.Length)
		}

		if packet.IsClose {
			session.HandleClose()
			sessions.Invalidate(packet.Session)
		}
	})
}

type Packet struct {
	Tokens  [][]byte
	Err     error
	Session int

	IsConnect bool

	IsData bool
	Pos    int
	Data   []byte

	IsAck  bool
	Length int

	IsClose bool
}

func ParsePacket(bs []byte) (*Packet, error) {
	if len(bs) == 0 || bs[0] != '/' || bs[len(bs)-1] != '/' {
		// This is an illegal packet, fail to parse.
		return nil, errors.New("malformed packet")
	}

	p := &Packet{Tokens: Split(bs)}
	kind := string(p.ReadBytes())
	p.Session = p.ReadInt()

	if kind == "connect" {
		p.IsConnect = true
	}

	if kind == "data" {
		p.IsData = true
		p.Pos = p.ReadInt()
		p.Data = p.ReadBytes()
	}

	if kind == "ack" {
		p.IsAck = true
		p.Length = p.ReadInt()
	}

	if kind == "close" {
		p.IsClose = true
	}

	if p.Err == nil && len(p.Tokens) > 0 {
		p.Err = errors.New("too many tokens")
	}

	return p, p.Err
}

func (p *Packet) ReadBytes() []byte {
	var bs []byte
	if p.Err == nil {
		if len(p.Tokens) == 0 {
			p.Err = errors.New("out of tokens")
			return bs
		}

		bs, p.Tokens = p.Tokens[0], p.Tokens[1:]
	}
	return bs
}

func (p *Packet) ReadInt() int {
	bs := p.ReadBytes()

	var n int64
	if p.Err == nil {
		n, p.Err = strconv.ParseInt(string(bs), 10, 0)
	}
	if p.Err == nil && (n < 0 || n >= 2147483648) {
		p.Err = errors.New("out of bounds")
	}
	return int(n)
}
