package main

import (
	"bufio"
	"encoding/json"
	"github.com/bbeck/protohackers/internal"
	"io"
	"math"
	"net"
)

type Request struct {
	Method *string  `json:"method"`
	Number *float64 `json:"number"`
}

const Prime = `{"method":"isPrime","prime":true}` + "\n"
const NotPrime = `{"method":"isPrime","prime":false}` + "\n"
const Malformed = `{"method":"malformed"}` + "\n"

func main() {
	internal.RunTCPServer(func(conn net.Conn) {
		defer conn.Close()

		r := bufio.NewReaderSize(conn, 1024*1024)
		for {
			bs, _, err := r.ReadLine()
			if err != nil {
				io.WriteString(conn, Malformed)
				return
			}

			var request Request
			if err := json.Unmarshal(bs, &request); err != nil || IsMalformed(request) {
				io.WriteString(conn, Malformed)
				return
			}

			if IsPrime(*request.Number) {
				io.WriteString(conn, Prime)
			} else {
				io.WriteString(conn, NotPrime)
			}
		}
	})
}

func IsMalformed(r Request) bool {
	if r.Method == nil || *r.Method != "isPrime" || r.Number == nil {
		return true
	}
	if r.Number == nil {
		return true
	}
	return false
}

func IsPrime(n float64) bool {
	if _, frac := math.Modf(n); frac != 0. {
		return false
	}

	if n <= 1. {
		return false
	}

	if n <= 3. {
		return true
	}

	sqrt := math.Sqrt(n)
	for i := 2.; i <= sqrt; i++ {
		if math.Mod(n, i) == 0. {
			return false
		}
	}

	return true
}
