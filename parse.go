package main

import (
	"strings"
	"unicode"
)

func parseKarma(reader *strings.Reader) int {
	var err error
	net := 0

	ch, _, err := reader.ReadRune()
	for err == nil && (ch == '+' || ch == '-') {
		next, _, err := reader.ReadRune()
		if err == nil {
			if ch == '+' && next == '+' {
				net++
			}
			if ch == '-' && next == '-' {
				net--
			}
		}

		ch, _, err = reader.ReadRune()
	}

	return net
}

func parseSubjectTilWhitespaceOrKarma(reader *strings.Reader) (string, bool) {
	subj := strings.Builder{}
	ch, _, err := reader.ReadRune()

	shouldStop := func(r rune) bool {
		return r == '+' || r == '-' || unicode.IsSpace(r)
	}

	for err == nil && !shouldStop(ch) {
		subj.WriteRune(ch)
		ch, _, err = reader.ReadRune()
	}

	if shouldStop(ch) {
		// Put back the character we took off that caused us to stop
		_ = reader.UnreadRune()
	}

	s := subj.String()

	return s, len(s) > 0
}

func parseSubjectInParens(reader *strings.Reader) (string, bool) {
	subj := strings.Builder{}

	ch, _, err := reader.ReadRune()
	if err != nil || ch != '(' {
		return "", false
	}

	ch, _, err = reader.ReadRune()
	for err == nil && ch != ')' {
		subj.WriteRune(ch)

		ch, _, err = reader.ReadRune()
	}

	if ch != ')' {
		return "", false
	}

	s := subj.String()

	return s, len(s) > 0
}

func parseModifier(reader *strings.Reader) (string, int, bool) {
	var subj string
	ok := false

	ch, _, err := reader.ReadRune()
	if err != nil {
		return "", 0, false
	}

	// safe because this is 1 unread for 1 read
	_ = reader.UnreadRune()

	if ch == '(' {
		subj, ok = parseSubjectInParens(reader)
	} else {
		subj, ok = parseSubjectTilWhitespaceOrKarma(reader)
	}

	karma := parseKarma(reader)

	return subj, karma, ok
}

func ParseModifiers(s string) map[string]int {
	var err error

	subjects := make(map[string]int)

	reader := strings.NewReader(s)
	for err == nil {
		ch, _, err := reader.ReadRune()
		// end of string, all done
		if err != nil {
			break
		}

		if !unicode.IsSpace(ch) {
			// Safe because this is 1 unread for 1 read
			_ = reader.UnreadRune()

			subj, karma, ok := parseModifier(reader)
			if !ok {
				continue
			}

			subjects[subj] += karma
		}
	}

	return subjects
}
