package main

import (
	"bufio"
	"github.com/bbeck/protohackers/internal"
	"net"
	"strconv"
	"strings"
)

func main() {
	internal.RunTCPServer(func(conn net.Conn) {
		defer conn.Close()

		r := bufio.NewReader(conn)
		header, err := r.ReadBytes(0)
		if err != nil {
			return
		}

		encrypt, decrypt := GetCiphers(header)
		if IsIdentityCipher(encrypt) {
			return
		}

		// Re-create the ciphers so that any state that was mutated by identity
		// checking is reset.
		encrypt, decrypt = GetCiphers(header)

		// Wrap the reader in a decrypter, and build an encrypting writer.
		in := bufio.NewScanner(&Decrypter{r: r, ciphers: decrypt})
		out := Encrypter{w: conn, ciphers: encrypt}

		for in.Scan() {
			if in.Err() != nil {
				return
			}

			line := in.Text()
			choice := ChooseToy(line)
			out.Write([]byte(choice + "\n"))
		}
	})
}

func GetCiphers(bs []byte) ([]CipherFunc, []CipherFunc) {
	var encrypt, decrypt []CipherFunc

	for i := 0; i < len(bs); i++ {
		switch bs[i] {
		case 0x01:
			encrypt = append(encrypt, ReverseBits())
			decrypt = append(decrypt, ReverseBits())

		case 0x02:
			N := bs[i+1]
			encrypt = append(encrypt, XorN(N))
			decrypt = append(decrypt, XorN(N))
			i++

		case 0x03:
			encrypt = append(encrypt, XorPos())
			decrypt = append(decrypt, XorPos())

		case 0x04:
			N := bs[i+1]
			encrypt = append(encrypt, AddN(N))
			decrypt = append(decrypt, SubN(N))
			i++

		case 0x05:
			encrypt = append(encrypt, AddPos())
			decrypt = append(decrypt, SubPos())
		}
	}

	return encrypt, decrypt
}

func IsIdentityCipher(ciphers []CipherFunc) bool {
	for n := 0; n < 10; n++ {
		for b := 0; b < 256; b++ {
			data := byte(b)
			for _, cipher := range ciphers {
				data = cipher(data)
			}
			if data != byte(b) {
				return false
			}
		}
	}
	return true
}

func ChooseToy(s string) string {
	var count int64
	var best string
	for _, part := range strings.Split(s, ",") {
		n, _, _ := strings.Cut(part, "x")
		c, _ := strconv.ParseInt(n, 10, 64)
		if c > count {
			count = c
			best = part
		}
	}

	return best
}
