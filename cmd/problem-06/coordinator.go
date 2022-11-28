package main

import (
	"math"
	"sync"
)

type Coordinator struct {
	sync.Mutex
	Clients            map[int]*Client
	Observations       map[string][]Observation
	Limits             map[Road]uint16
	SentTickets        map[string][]uint32
	TicketsToSendLater map[Road][]Ticket
}

func (c *Coordinator) AddClient(client *Client) {
	c.Lock()
	defer c.Unlock()
	c.Clients[client.ID] = client

	if client.IsCamera {
		c.Limits[client.Road] = client.Limit
	}

	if client.IsDispatcher {
		for _, road := range client.Roads {
			for _, ticket := range c.TicketsToSendLater[road] {
				client.WriteTicket(ticket)
			}
			delete(c.TicketsToSendLater, road)
		}
	}
}

func (c *Coordinator) AddPlate(plate string, tm uint32, road Road, mile uint16) {
	c.Lock()
	defer c.Unlock()

	observation := Observation{
		Plate:     plate,
		Timestamp: tm,
		Road:      road,
		Mile:      mile,
	}
	c.Observations[plate] = append(c.Observations[plate], observation)

	// See if this observation leads to any tickets
	for _, other := range c.Observations[plate] {
		if observation.Road != other.Road || observation.Timestamp == other.Timestamp {
			continue
		}

		ds := Abs(int(observation.Mile) - int(other.Mile))
		dt := Abs(int(observation.Timestamp) - int(other.Timestamp))
		speed := math.Round(3600 * float64(ds) / float64(dt))
		limit := float64(c.Limits[observation.Road])
		if speed <= limit {
			continue
		}

		// This generates a ticket
		var earlier, later Observation
		if observation.Timestamp < other.Timestamp {
			earlier = observation
			later = other
		} else {
			earlier = other
			later = observation
		}
		c.SendTicket(Ticket{
			Plate:      observation.Plate,
			Road:       observation.Road,
			Timestamp1: earlier.Timestamp,
			Timestamp2: later.Timestamp,
			Mile1:      earlier.Mile,
			Mile2:      later.Mile,
			Speed:      uint16(100 * speed),
		})
	}
}

func (c *Coordinator) SendTicket(t Ticket) {
	// NOTE: Don't lock/unlock, this is called with the mutex already acquired.

	// Don't send a 2nd ticket for this day
	day1 := t.Timestamp1 / 86400
	day2 := t.Timestamp2 / 86400
	contains1 := Contains(c.SentTickets[t.Plate], day1)
	contains2 := Contains(c.SentTickets[t.Plate], day2)
	if contains1 || contains2 {
		return
	}

	if !contains1 {
		c.SentTickets[t.Plate] = append(c.SentTickets[t.Plate], day1)
	}
	if !contains2 {
		c.SentTickets[t.Plate] = append(c.SentTickets[t.Plate], day2)
	}

	// Find a dispatcher for this ticket's road.
	var dispatcher *Client
	for _, client := range c.Clients {
		if client.IsDispatcher && Contains(client.Roads, t.Road) {
			dispatcher = client
			break
		}
	}

	if dispatcher == nil {
		c.TicketsToSendLater[t.Road] = append(c.TicketsToSendLater[t.Road], t)
		return
	}

	// Send this ticket now
	dispatcher.WriteTicket(t)
}

func (c *Coordinator) Remove(client *Client) {
	c.Lock()
	defer c.Unlock()

	delete(c.Clients, client.ID)
}

func (c *Coordinator) SendHeartbeats() {
	c.Lock()
	defer c.Unlock()

	for _, client := range c.Clients {
		if !client.WantsHeartbeat {
			continue
		}

		client.HeartbeatCounter--
		if client.HeartbeatCounter == 0 {
			client.HeartbeatCounter = client.HeartbeatInterval
			client.Write8(0x41)
		}
	}
}

type Observation struct {
	Plate     string
	Timestamp uint32
	Road      Road
	Mile      uint16
}

type Ticket struct {
	Plate                  string
	Road                   Road
	Timestamp1, Timestamp2 uint32
	Mile1, Mile2           uint16
	Speed                  uint16
}
