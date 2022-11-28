package main

import (
	"encoding/binary"
	"github.com/bbeck/protohackers/internal"
	"net"
	"sync"
	"time"
)

func main() {
	coordinator := Coordinator{
		Clients:            make(map[int]*Client),
		Observations:       make(map[string][]Observation),
		Limits:             make(map[Road]uint16),
		SentTickets:        make(map[string][]uint32),
		TicketsToSendLater: make(map[Road][]Ticket),
	}
	go SendHeartbeats(&coordinator)

	internal.RunTCPServer(func(conn net.Conn) {
		client := &Client{ID: GetNextID(), Connection: conn}

		defer func() {
			coordinator.Remove(client)
			conn.Close()
		}()

		for client.Err == nil {
			switch client.Read8() {
			case 0x20: // Plate
				if client.IsDispatcher {
					client.WriteError("illegal plate message from dispatcher")
					return
				}

				plate := client.ReadString()
				tm := client.Read32()
				coordinator.AddPlate(plate, tm, client.Road, client.Mile)

			case 0x40: // WantHeartbeat
				client.HeartbeatInterval = client.Read32()
				client.HeartbeatCounter = client.HeartbeatInterval
				client.WantsHeartbeat = client.HeartbeatInterval > 0
				coordinator.AddClient(client)

			case 0x80: // IAmCamera
				if client.IsCamera || client.IsDispatcher {
					client.WriteError("illegal IAmCamera message")
					return
				}

				client.IsCamera = true
				client.Road = Road(client.Read16())
				client.Mile = client.Read16()
				client.Limit = client.Read16()
				coordinator.AddClient(client)

			case 0x81: // IAmDispatcher
				if client.IsCamera || client.IsDispatcher {
					client.WriteError("illegal IAmDispatcher message")
					return
				}

				client.IsDispatcher = true
				client.Roads = make([]Road, client.Read8())
				for i := 0; i < len(client.Roads); i++ {
					client.Roads[i] = Road(client.Read16())
				}
				coordinator.AddClient(client)

			default:
				client.WriteError("unsupported message")
				return
			}
		}
	})
}

type Road uint16

type Client struct {
	ID         int
	Connection net.Conn
	Err        error

	WantsHeartbeat    bool
	HeartbeatInterval uint32
	HeartbeatCounter  uint32

	IsDispatcher bool
	Roads        []Road

	IsCamera    bool
	Road        Road
	Mile, Limit uint16
}

func (c *Client) Read8() uint8 {
	var data uint8
	if c.Err == nil {
		c.Err = binary.Read(c.Connection, binary.BigEndian, &data)
	}
	return data
}

func (c *Client) Read16() uint16 {
	var data uint16
	if c.Err == nil {
		c.Err = binary.Read(c.Connection, binary.BigEndian, &data)
	}
	return data
}

func (c *Client) Read32() uint32 {
	var data uint32
	if c.Err == nil {
		c.Err = binary.Read(c.Connection, binary.BigEndian, &data)
	}
	return data
}

func (c *Client) ReadString() string {
	data := make([]byte, c.Read8())
	if c.Err == nil {
		c.Err = binary.Read(c.Connection, binary.BigEndian, &data)
	}
	return string(data)
}

func (c *Client) Write8(n uint8) {
	if c.Err == nil {
		c.Err = binary.Write(c.Connection, binary.BigEndian, n)
	}
}

func (c *Client) Write16(n uint16) {
	if c.Err == nil {
		c.Err = binary.Write(c.Connection, binary.BigEndian, n)
	}
}

func (c *Client) Write32(n uint32) {
	if c.Err == nil {
		c.Err = binary.Write(c.Connection, binary.BigEndian, n)
	}
}

func (c *Client) WriteString(s string) {
	c.Write8(uint8(len(s)))
	if c.Err == nil {
		c.Err = binary.Write(c.Connection, binary.BigEndian, []byte(s))
	}
}

func (c *Client) WriteError(s string) {
	c.Write8(0x10)
	c.WriteString(s)
}

func (c *Client) WriteTicket(t Ticket) {
	c.Write8(0x21)
	c.WriteString(t.Plate)
	c.Write16(uint16(t.Road))
	c.Write16(t.Mile1)
	c.Write32(t.Timestamp1)
	c.Write16(t.Mile2)
	c.Write32(t.Timestamp2)
	c.Write16(t.Speed)
}

var NextID int
var NextIDMutex sync.Mutex

func GetNextID() int {
	NextIDMutex.Lock()
	defer NextIDMutex.Unlock()

	id := NextID
	NextID++
	return id
}

func SendHeartbeats(c *Coordinator) {
	for range time.Tick(time.Second / 10) {
		c.SendHeartbeats()
	}
}
