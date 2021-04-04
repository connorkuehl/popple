package main

type Modifier struct {
	Name     string
	NetKarma int
}

func ParseModifier(s string) *Modifier {
	runes := []rune(s)
	nameSpan := FindName(runes)
	name := string(runes[nameSpan.start : nameSpan.start+nameSpan.count])
	if name == "" {
		return nil
	}

	netKarma := ParseNetKarma(runes[nameSpan.start+nameSpan.count:])
	if netKarma == 0 {
		return nil
	}

	return &Modifier{name, netKarma}
}
