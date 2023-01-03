package main

func Split(in []byte) [][]byte {
	var fields [][]byte

	var buf []byte
	var escaped bool
	for pos := 0; pos < len(in); pos++ {
		if !escaped && in[pos] == '\\' {
			escaped = true
			continue
		}

		if escaped || in[pos] != '/' {
			buf = append(buf, in[pos])
		} else {
			if len(buf) > 0 {
				fields = append(fields, buf)
			}
			buf = nil
		}

		escaped = false
	}

	if len(buf) > 0 {
		fields = append(fields, buf)
	}

	return fields
}

func Reverse(in []byte) []byte {
	N := len(in)
	out := make([]byte, N)
	for i := 0; i < len(in); i++ {
		out[i] = in[N-i-1]
	}

	return out
}

func Escape(in []byte) []byte {
	var out []byte
	for _, b := range in {
		if b == '/' || b == '\\' {
			out = append(out, '\\')
		}
		out = append(out, b)
	}
	return out
}

func Min(ns ...int) int {
	min := ns[0]
	for _, n := range ns[1:] {
		if n < min {
			min = n
		}
	}
	return min
}
