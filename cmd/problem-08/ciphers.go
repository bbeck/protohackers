package main

import (
	"io"
	"math/bits"
)

type CipherFunc func(byte) byte

// ReverseBits cipher.
//
// Reverse the order of bits in the byte, so the least-significant bit becomes
// the most-significant bit, the 2nd-least-significant becomes the
// 2nd-most-significant, and so on.
func ReverseBits() CipherFunc {
	return func(b byte) byte {
		return bits.Reverse8(b)
	}
}

// XorN cipher.
//
// XOR the byte by the value N. Note that 0 is a valid value for N.
func XorN(n byte) CipherFunc {
	return func(b byte) byte {
		return b ^ n
	}
}

// XorPos cipher.
//
// XOR the byte by its position in the stream, starting from 0.
func XorPos() CipherFunc {
	var pos byte
	return func(b byte) byte {
		b ^= pos
		pos++
		return b
	}
}

// AddN cipher.
//
// Add N to the byte, modulo 256. Note that 0 is a valid value for N, and
// addition wraps, so that 255+1=0, 255+2=1, and so on.
func AddN(n byte) CipherFunc {
	return func(b byte) byte {
		return b + n
	}
}

// SubN cipher.
func SubN(n byte) CipherFunc {
	return func(b byte) byte {
		return b - n
	}
}

// AddPos cipher.
//
// Add the position in the stream to the byte, modulo 256, starting from 0.
// Addition wraps, so that 255+1=0, 255+2=1, and so on.
func AddPos() CipherFunc {
	var pos byte
	return func(b byte) byte {
		b += pos
		pos++
		return b
	}
}

// SubPos cipher.
func SubPos() CipherFunc {
	var pos byte
	return func(b byte) byte {
		b -= pos
		pos++
		return b
	}
}

type Encrypter struct {
	w       io.Writer
	ciphers []CipherFunc
}

func (e *Encrypter) Write(bs []byte) (int, error) {
	for i := 0; i < len(bs); i++ {
		for _, cipher := range e.ciphers {
			bs[i] = cipher(bs[i])
		}
	}

	n, err := e.w.Write(bs)
	return n, err
}

type Decrypter struct {
	r       io.Reader
	ciphers []CipherFunc
}

func (d *Decrypter) Read(bs []byte) (int, error) {
	n, err := d.r.Read(bs)
	for i := 0; i < n; i++ {
		for j := len(d.ciphers) - 1; j >= 0; j-- {
			bs[i] = d.ciphers[j](bs[i])
		}
	}

	return n, err
}
