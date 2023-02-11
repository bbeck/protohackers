package main

import (
	"net"
	"sync"
)

type Authorities struct {
	sync.Mutex
	Cache map[uint32]*Authority
}

func NewAuthorities() *Authorities {
	return &Authorities{
		Cache: make(map[uint32]*Authority),
	}
}

func (a *Authorities) GetAuthority(site uint32) (*Authority, error) {
	a.Lock()
	defer a.Unlock()

	if authority := a.Cache[site]; authority != nil {
		return authority, nil
	}

	authority, err := NewAuthority(site)
	if err != nil {
		return nil, err
	}

	a.Cache[site] = authority
	return authority, nil
}

const AuthorityAddress = "pestcontrol.protohackers.com:20547"

type Authority struct {
	Channel    chan *SiteVisit
	Connection net.Conn
	Targets    map[Species]Target
	Actions    map[Species]Action
	PolicyIDs  map[Species]uint32
}

func NewAuthority(site uint32) (*Authority, error) {
	conn, err := net.Dial("tcp", AuthorityAddress)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	// Send Hello
	if err := WriteMessage(conn, HelloMessage{}); err != nil {
		_ = conn.Close()
		return nil, err
	}

	// Receive Hello
	if _, err := ReadMessage[HelloMessage](conn); err != nil {
		_ = WriteMessage(conn, Error{Message: err.Error()})
		_ = conn.Close()
		return nil, err
	}

	// Get the target populations for this site.
	if err := WriteMessage(conn, DialAuthority{Site: site}); err != nil {
		_ = conn.Close()
		return nil, err
	}

	target, err := ReadMessage[TargetPopulations](conn)
	if err != nil {
		_ = WriteMessage(conn, Error{Message: err.Error()})
		_ = conn.Close()
		return nil, err
	}

	authority := &Authority{
		Channel:    make(chan *SiteVisit, 10),
		Connection: conn,
		Targets:    target.Populations,
		Actions:    make(map[Species]Action),
		PolicyIDs:  make(map[Species]uint32),
	}

	go func(ch chan *SiteVisit) {
		for {
			authority.HandleRequest(<-ch)
		}
	}(authority.Channel)

	return authority, nil
}

func (a *Authority) HandleSiteVisit(sv *SiteVisit) {
	a.Channel <- sv
}

func (a *Authority) HandleRequest(sv *SiteVisit) {
	// Ensure there's a population reported for each species we have a target
	// for.  If a species is missing from the population report then treat it as
	// a zero.
	for species := range a.Targets {
		if _, found := sv.Populations[species]; !found {
			sv.Populations[species] = 0
		}
	}

	// Determine which policies to put into effect.
	for species, target := range a.Targets {
		count := sv.Populations[species]

		var err error
		if count < target.Min {
			err = a.SetPolicy(species, ConserveAction)
		} else if count > target.Max {
			err = a.SetPolicy(species, CullAction)
		} else {
			err = a.SetPolicy(species, DoNothingAction)
		}

		if err != nil {
			return
		}
	}
}

func (a *Authority) SetPolicy(species Species, action Action) error {
	if a.Actions[species] == action {
		// We're not changing the current action, there's nothing to do.
		return nil
	}

	// Remove the existing policy if there is one.
	if id, found := a.PolicyIDs[species]; found {
		if err := WriteMessage(a.Connection, DeletePolicy{Policy: id}); err != nil {
			return err
		}

		if _, err := ReadMessage[OK](a.Connection); err != nil {
			_ = WriteMessage(a.Connection, Error{Message: err.Error()})
			return err
		}

		delete(a.Actions, species)
		delete(a.PolicyIDs, species)
	}

	// Add the new policy if there is one.
	if action != DoNothingAction {
		if err := WriteMessage(a.Connection, CreatePolicy{Species: species, Action: action}); err != nil {
			return err
		}

		m, err := ReadMessage[PolicyResult](a.Connection)
		if err != nil {
			_ = WriteMessage(a.Connection, Error{Message: err.Error()})
			return err
		}

		a.Actions[species] = action
		a.PolicyIDs[species] = m.Policy
	}

	return nil
}

type Species string

type Target struct {
	Min, Max uint32
}

const (
	DoNothingAction Action = 0x00
	CullAction      Action = 0x90
	ConserveAction  Action = 0xa0
)

type Action uint8

func (a Action) String() string {
	if a == DoNothingAction {
		return "do-nothing"
	}
	if a == CullAction {
		return "cull"
	}
	if a == ConserveAction {
		return "conserve"
	}
	return "?"
}
