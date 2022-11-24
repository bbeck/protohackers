package main

import (
	"encoding/binary"
	"github.com/bbeck/protohackers/internal"
	"net"
)

type Price struct {
	Timestamp, Price int32
}

func main() {
	internal.RunWithTunnel(func(conn net.Conn) {
		var err error
		read := func(data ...any) {
			for i := 0; err == nil && i < len(data); i++ {
				err = binary.Read(conn, binary.BigEndian, data[i])
			}
		}

		write := func(data ...any) {
			for i := 0; err == nil && i < len(data); i++ {
				err = binary.Write(conn, binary.BigEndian, data[i])
			}
		}

		defer conn.Close()

		var prices []Price
		for {
			var kind byte
			var a, b int32
			read(&kind, &a, &b)

			switch kind {
			case 'I':
				prices = append(prices, Price{Timestamp: a, Price: b})

			case 'Q':
				var sum, count int
				for _, p := range prices {
					if a <= p.Timestamp && p.Timestamp <= b {
						sum += int(p.Price)
						count++
					}
				}

				if count != 0 {
					write(int32(sum / count))
				} else {
					write(int32(0))
				}

			default:
				return
			}
		}
	})
}
