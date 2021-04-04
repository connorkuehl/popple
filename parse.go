package main

import "unicode"

func ParseNetKarma(runes []rune) int {
	net := 0
	len := len(runes)

	for i := 1; i < len; i += 2 {
		if runes[i-1] == '+' && runes[i] == '+' {
			net++
		} else if runes[i-1] == '-' && runes[i] == '-' {
			net--
		} else {
			break
		}
	}

	return net
}

func FindName(runes []rune) *Span {
	len := len(runes)
	start := 0
	count := 0

	if len > 0 && runes[0] == '@' {
		start++
	}
	for i := start; i < len; i++ {
		if unicode.IsLetter(runes[i]) {
			count++
		} else {
			break
		}
	}

	return &Span{start, count}
}

func ParseName(s string) string {
	runes := []rune(s)
	sp := FindName(runes)
	return string(runes[sp.start : sp.start+sp.count])
}
