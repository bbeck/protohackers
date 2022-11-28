package main

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func Abs[T Integer](n T) T {
	if n < 0 {
		return -n
	}
	return n
}

func Min[T Integer](elems ...T) T {
	min := elems[0]
	for _, elem := range elems[1:] {
		if elem < min {
			min = elem
		}
	}
	return min
}

func Max[T Integer](elems ...T) T {
	max := elems[0]
	for _, elem := range elems[1:] {
		if elem > max {
			max = elem
		}
	}
	return max
}

func Contains[T comparable](a []T, elem T) bool {
	for _, e := range a {
		if e == elem {
			return true
		}
	}
	return false
}
