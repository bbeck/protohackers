package main

import (
	"fmt"
	"github.com/bbeck/protohackers/internal"
	"strings"
)

func main() {
	db := map[string]string{
		"version": "alpha",
	}

	internal.RunUDPServer(func(bs []byte) []byte {
		s := string(bs)

		if key, value, found := strings.Cut(s, "="); found {
			if key != "version" {
				db[key] = value
			}
			return nil
		}

		return []byte(fmt.Sprintf("%s=%s", s, db[s]))
	})
}
