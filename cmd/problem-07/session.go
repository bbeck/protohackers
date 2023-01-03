package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	SessionExpiration     = 60 * time.Second
	RetransmissionTimeout = 3 * time.Second
)

type Sessions struct {
	sync.Mutex

	Cache      map[int]*Session
	LastAccess map[int]time.Time
}

func NewSessions() *Sessions {
	sessions := &Sessions{
		Cache:      make(map[int]*Session),
		LastAccess: make(map[int]time.Time),
	}

	// Create the background reaper process that closes expired sessions and
	// removes them from the cache.
	go func() {
		for range time.Tick(SessionExpiration / 10) {
			sessions.Mutex.Lock()

			for id, last := range sessions.LastAccess {
				if time.Now().Sub(last) >= SessionExpiration {
					session := sessions.Cache[id]
					if !session.Closed {
						session.HandleClose()
					}
					delete(sessions.Cache, id)
					delete(sessions.LastAccess, id)
				}
			}

			sessions.Mutex.Unlock()
		}
	}()

	return sessions
}

func (s *Sessions) Get(id int) *Session {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	session, present := s.Cache[id]
	if !present {
		return nil
	}
	s.LastAccess[id] = time.Now()
	return session
}

func (s *Sessions) Put(id int, session *Session) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	s.Cache[id] = session
	s.LastAccess[id] = time.Now()
}

func (s *Sessions) Invalidate(id int) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	delete(s.Cache, id)
	delete(s.LastAccess, id)
}

// =============================================================================

type Session struct {
	sync.Mutex

	ID          int
	Closed      bool
	Send        func([]byte)
	Application *Application

	// The position in the stream that we've completely received.
	ReceivedTo int

	Buffer       []byte    // The buffer of data to send
	SentTo       int       // The last position in the buffer we've sent
	LastSendTime time.Time // The time we last sent data
	AckTo        int       // The last position we've received an ack for
}

func NewSession(id int, send func([]byte)) *Session {
	session := &Session{
		ID:   id,
		Send: send,
	}
	session.Application = &Application{Session: session}

	// Create the background goroutine to retransmit messages that aren't acked in
	// a timely fashion.  This goroutine will stop when the session is closed.
	go func() {
		for range time.Tick(RetransmissionTimeout / 10) {
			if session.Closed {
				break
			}

			session.Lock()
			if time.Now().Sub(session.LastSendTime) > RetransmissionTimeout {
				session.SentTo = session.AckTo
				session.MaybeSend()
			}
			session.Unlock()
		}
	}()

	return session
}

func (s *Session) Reply(kind string, args ...any) {
	switch kind {
	case "ack":
		s.Send([]byte(fmt.Sprintf("/ack/%d/%d/", s.ID, args[0])))
	case "close":
		s.Send([]byte(fmt.Sprintf("/close/%d/", s.ID)))
	case "data":
		s.Send([]byte(fmt.Sprintf("/data/%d/%d/%s/", s.ID, args[0], args[1])))
	}
}

func (s *Session) HandleConnect() {
	s.Lock()
	defer s.Unlock()

	// Send /ack/SESSION/0/ to let the client know that the session is open
	// (do this even if it is a duplicate connect, because the first ack may
	// have been dropped).
	s.Reply("ack", s.ReceivedTo)
}

func (s *Session) HandleData(pos int, data []byte) {
	s.Lock()
	defer s.Unlock()

	// If you have not received everything up to POS: send a duplicate of your
	// previous ack (or /ack/SESSION/0/ if none), saying how much you have
	// received, to provoke the other side to retransmit whatever you're
	// missing.
	if s.ReceivedTo < pos {
		s.Reply("ack", s.ReceivedTo)
		return
	}

	// If you've already received everything up to POS: find the total LENGTH of
	// unescaped data that you've already received (including the data in this
	// message, if any), send /ack/SESSION/LENGTH/, and pass on the new data (if
	// any) to the application layer.
	start := Min(s.ReceivedTo-pos, len(data))
	data = data[start:]

	s.ReceivedTo += len(data)
	s.Reply("ack", s.ReceivedTo)
	s.Application.Write(data)
}

func (s *Session) HandleAck(length int) {
	s.Lock()
	defer s.Unlock()

	// If the LENGTH value is not larger than the largest LENGTH value in any ack
	// message you've received on this session so far: do nothing and stop (assume
	// it's a duplicate ack that got delayed).
	if length < s.AckTo {
		return
	}

	// If the LENGTH value is larger than the total amount of payload you've sent:
	// the peer is misbehaving, close the session.
	if length > s.SentTo {
		s.Close()
		return
	}

	// If the LENGTH value is smaller than the total amount of payload you've
	// sent: retransmit all payload data after the first LENGTH bytes.

	// If the LENGTH value is equal to the total amount of payload you've sent:
	// don't send any reply.
	s.AckTo = length
	s.MaybeSend()
}

func (s *Session) HandleClose() {
	s.Lock()
	defer s.Unlock()

	s.Close()
}

func (s *Session) Close() {
	// When you receive a /close/SESSION/ message, send a matching close
	// message back.
	s.Reply("close")

	s.Closed = true
}

func (s *Session) Write(bs []byte) {
	// Lock is held by caller

	s.Buffer = append(s.Buffer, bs...)
	s.MaybeSend()
}

func (s *Session) MaybeSend() {
	for s.SentTo < len(s.Buffer) {
		// Process the tail of the buffer in 500 byte increments.  This ensures we
		// never commit to sending more data than is allowed in a single message.
		end := Min(s.SentTo+500, len(s.Buffer))
		toSend := s.Buffer[s.SentTo:end]
		s.Reply("data", s.SentTo, Escape(toSend))
		s.SentTo += len(toSend)
		s.LastSendTime = time.Now()
	}
}

// =============================================================================

type Application struct {
	Session *Session
	Buffer  []byte
}

func (a *Application) Write(bs []byte) {
	for _, b := range bs {
		if b != '\n' {
			a.Buffer = append(a.Buffer, b)
			continue
		}

		line := Reverse(a.Buffer)
		line = append(line, '\n')
		a.Session.Write(line)
		a.Buffer = nil
	}
}
