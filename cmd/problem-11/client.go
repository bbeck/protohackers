package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type Readable[T any] interface {
	Type() uint8
	Unmarshal([]byte) (*T, error)
}

func ReadMessage[T Readable[T]](r io.Reader) (*T, error) {
	var message T
	var kind uint8
	var length uint32
	var payload []byte
	var checksum uint8
	var err error

	kind, err = Read8(r)
	if err != nil {
		return nil, err
	}

	if kind == 0x51 {
		// We received an error message instead of our desired message type.
		_, _ = Read32(r) // length
		var msg string
		msg, err = ReadString(r)
		_, _ = Read8(r) // checksum

		return nil, errors.New(fmt.Sprintf("received error: %s", msg))
	}

	if message.Type() != kind {
		return nil, errors.New(fmt.Sprintf("unexpected message type: expected=%x, received=%x", message.Type(), kind))
	}

	length, err = Read32(r)
	if err != nil {
		return nil, err
	}
	if length > 1e6 {
		return nil, errors.New("length too long")
	}

	// Remove 6 bytes that are held outside the payload:
	//   - type     (1 byte)
	//   - length   (4 bytes)
	//   - checksum (1 byte)
	payload, err = ReadBytes(r, length-6)
	if err != nil {
		return nil, err
	}
	if len(payload) != int(length-6) {
		return nil, errors.New("too few bytes read")
	}

	checksum, err = Read8(r)
	if err != nil {
		return nil, err
	}

	// Validate the checksum
	var sum int
	sum += int(kind)
	sum += int((length >> 24) & 0xFF)
	sum += int((length >> 16) & 0xFF)
	sum += int((length >> 8) & 0xFF)
	sum += int((length >> 0) & 0xFF)
	for _, b := range payload {
		sum += int(b)
	}
	sum += int(checksum)

	if sum%256 != 0 {
		return nil, errors.New(fmt.Sprintf("invalid checksum: %x (decimal: %d)", sum, sum))
	}

	// Unmarshall the payload into a message
	return message.Unmarshal(payload)
}

type Writable interface {
	Type() uint8
	Marshal() []byte
}

func WriteMessage[T Writable](w io.Writer, message T) error {
	if err := Write8(w, message.Type()); err != nil {
		return err
	}

	payload := message.Marshal()

	// Add 6 bytes due to the type (1 byte), length (4 bytes) and
	// checksum (1 byte) fields.
	length := uint32(len(payload) + 6)
	if err := Write32(w, length); err != nil {
		return err
	}

	if err := WriteBytes(w, payload); err != nil {
		return err
	}

	var sum uint8
	sum += message.Type()
	sum += uint8((length >> 24) & 0xFF)
	sum += uint8((length >> 16) & 0xFF)
	sum += uint8((length >> 8) & 0xFF)
	sum += uint8((length >> 0) & 0xFF)
	for _, b := range payload {
		sum += b
	}

	checksum := uint8(256 - int(sum))
	if err := Write8(w, checksum); err != nil {
		return err
	}

	return nil
}

// /////////////////////////////////////////////////////////////////////////////

type HelloMessage struct{}

func (m HelloMessage) Type() uint8 {
	return 0x50
}

func (m HelloMessage) Marshal() []byte {
	var w bytes.Buffer
	_ = WriteString(&w, "pestcontrol")
	_ = Write32(&w, 1)
	return w.Bytes()
}

func (m HelloMessage) Unmarshal(data []byte) (*HelloMessage, error) {
	r := bytes.NewReader(data)

	protocol, err := ReadString(r)
	if err != nil {
		return nil, err
	}

	version, err := Read32(r)
	if err != nil {
		return nil, err
	}

	if protocol != "pestcontrol" {
		return nil, errors.New("invalid protocol")
	}

	if version != 1 {
		return nil, errors.New("invalid version")
	}

	if r.Len() > 0 {
		return nil, errors.New("not all bytes read")
	}

	return &HelloMessage{}, nil
}

// /////////////////////////////////////////////////////////////////////////////

type Error struct {
	Message string
}

func (m Error) Type() uint8 {
	return 0x51
}

func (m Error) Marshal() []byte {
	var w bytes.Buffer
	_ = WriteString(&w, m.Message)
	return w.Bytes()
}

// /////////////////////////////////////////////////////////////////////////////

type OK struct{}

func (m OK) Type() uint8 {
	return 0x52
}

func (m OK) Unmarshal(data []byte) (*OK, error) {
	if len(data) > 0 {
		return nil, errors.New("not all bytes read")
	}

	return &m, nil
}

// /////////////////////////////////////////////////////////////////////////////

type DialAuthority struct {
	Site uint32
}

func (m DialAuthority) Type() uint8 {
	return 0x53
}

func (m DialAuthority) Marshal() []byte {
	var w bytes.Buffer
	_ = Write32(&w, m.Site)
	return w.Bytes()
}

// /////////////////////////////////////////////////////////////////////////////

type TargetPopulations struct {
	Site        uint32
	Populations map[Species]Target
}

func (m TargetPopulations) Type() uint8 {
	return 0x54
}

func (m TargetPopulations) Unmarshal(data []byte) (*TargetPopulations, error) {
	var err error
	r := bytes.NewReader(data)

	m.Site, err = Read32(r)
	if err != nil {
		return nil, err
	}

	length, err := Read32(r)
	if err != nil {
		return nil, err
	}

	m.Populations = make(map[Species]Target)
	for n := uint32(0); n < length; n++ {
		s, err := ReadString(r)
		if err != nil {
			return nil, err
		}

		var target Target
		target.Min, err = Read32(r)
		if err != nil {
			return nil, err
		}

		target.Max, err = Read32(r)
		if err != nil {
			return nil, err
		}

		m.Populations[Species(s)] = target
	}

	if r.Len() > 0 {
		return nil, errors.New("not all bytes read")
	}

	return &m, nil
}

// /////////////////////////////////////////////////////////////////////////////

type CreatePolicy struct {
	Species Species
	Action  Action
}

func (m CreatePolicy) Type() uint8 {
	return 0x55
}

func (m CreatePolicy) Marshal() []byte {
	var w bytes.Buffer
	_ = WriteString(&w, string(m.Species))
	_ = Write8(&w, uint8(m.Action))
	return w.Bytes()
}

// /////////////////////////////////////////////////////////////////////////////

type DeletePolicy struct {
	Policy uint32
}

func (m DeletePolicy) Type() uint8 {
	return 0x56
}

func (m DeletePolicy) Marshal() []byte {
	var w bytes.Buffer
	_ = Write32(&w, m.Policy)
	return w.Bytes()
}

// /////////////////////////////////////////////////////////////////////////////

type PolicyResult struct {
	Policy uint32
}

func (m PolicyResult) Type() uint8 {
	return 0x57
}

func (m PolicyResult) Unmarshal(data []byte) (*PolicyResult, error) {
	var err error
	r := bytes.NewReader(data)

	m.Policy, err = Read32(r)
	if err != nil {
		return nil, err
	}

	if r.Len() > 0 {
		return nil, errors.New("not all bytes read")
	}

	return &m, nil
}

// /////////////////////////////////////////////////////////////////////////////

type SiteVisit struct {
	Site        uint32
	Populations map[Species]uint32
}

func (m SiteVisit) Type() uint8 {
	return 0x58
}

func (m SiteVisit) Unmarshal(data []byte) (*SiteVisit, error) {
	var err error
	r := bytes.NewReader(data)

	m.Site, err = Read32(r)
	if err != nil {
		return nil, err
	}

	length, err := Read32(r)
	if err != nil {
		return nil, err
	}

	m.Populations = make(map[Species]uint32)
	for n := uint32(0); n < length; n++ {
		s, err := ReadString(r)
		if err != nil {
			return nil, err
		}
		species := Species(s)

		count, err := Read32(r)
		if err != nil {
			return nil, err
		}

		if existing, found := m.Populations[species]; found && existing != count {
			return nil, errors.New("conflicting count")
		}

		m.Populations[species] = count
	}

	if r.Len() > 0 {
		return nil, errors.New("not all bytes read")
	}

	return &m, nil
}

// /////////////////////////////////////////////////////////////////////////////

func Read8(r io.Reader) (uint8, error) {
	var data uint8
	err := binary.Read(r, binary.BigEndian, &data)
	return data, err
}

func Read32(r io.Reader) (uint32, error) {
	var data uint32
	err := binary.Read(r, binary.BigEndian, &data)
	return data, err
}

func ReadArray[T any](r io.Reader) ([]T, error) {
	length, err := Read32(r)
	if err != nil {
		return nil, err
	}

	data := make([]T, length)
	err = binary.Read(r, binary.BigEndian, &data)
	return data, err
}

func ReadBytes(r io.Reader, length uint32) ([]byte, error) {
	data := make([]byte, length)
	err := binary.Read(r, binary.BigEndian, &data)
	return data, err
}

func ReadString(r io.Reader) (string, error) {
	bs, err := ReadArray[byte](r)
	return string(bs), err
}

func Write8(w io.Writer, data uint8) error {
	return binary.Write(w, binary.BigEndian, data)
}

func Write32(w io.Writer, data uint32) error {
	return binary.Write(w, binary.BigEndian, data)
}

func WriteBytes(w io.Writer, data []byte) error {
	return binary.Write(w, binary.BigEndian, data)
}

func WriteString(w io.Writer, data string) error {
	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, []byte(data))
}
