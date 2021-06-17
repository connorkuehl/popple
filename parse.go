package main

import (
	"strings"
	"unicode"
)

type Subject struct {
	Name  string
	Karma int
}

func ParseSubjects(s string) []Subject {
	subjects := make([]Subject, 0)
	var sub Subject
	var ok bool
	var remaining []rune

	remaining = []rune(s)
	for len(remaining) > 0 {
		sub, ok, remaining = tryParseSubject(remaining)
		if !ok || sub.Karma == 0 {
			continue
		}
		subjects = append(subjects, sub)
	}
	return subjects
}

func tryParseSubject(rs []rune) (Subject, bool, []rune) {
	remaining := seekSubjectStart(rs)
	/* in case an unclosed backtick forced the search to
	 * deplete the input stream, backtrack and try again,
	 * skipping over the unclosed backtick
	 */
	if len(remaining) == 0 {
		/* don't backtrack if the input was legitimately
		   completely consumed */
		if len(rs) > 1 && rs[0] == '`' && rs[len(rs)-1] == '`' {
			return Subject{}, false, []rune{}
		}
		return Subject{}, false, rs[1:]
	}

	var sub Subject
	var ok bool

	if remaining[0] == '(' {
		sub, ok, remaining = tryParseParens(remaining)
		/* if the parenthesis is not matched, the input
		 * stream will get depleted, so backtrack and try
		 * again, this time skipping over the opening paren
		 */
		if len(remaining) == 0 {
			return Subject{}, false, seekSubjectStart(rs[1:])
		}
	} else {
		sub, ok, remaining = tryParsePlain(remaining)
	}

	return sub, ok, remaining
}

type seekState int

const (
	seekingTick seekState = iota
	seekingWhitespace
	seekingStart
)

func seekSubjectStart(rs []rune) []rune {
	state := seekingStart
	start := 0
	for _, r := range rs {
		switch state {
		case seekingStart:
			if r != '`' && !unicode.IsSpace(r) {
				return rs[start:]
			}
			if r == '`' {
				state = seekingTick
			}
		case seekingTick:
			if r == '`' {
				state = seekingWhitespace
			}
		case seekingWhitespace:
			if unicode.IsSpace(r) {
				state = seekingStart
			}
			if r == '`' {
				state = seekingTick
			}
		}
		start++
	}
	return rs[start:]
}

func tryParseParens(rs []rune) (Subject, bool, []rune) {
	open := 1
	start := 1
	end := 1
	for _, r := range rs[start:] {
		if r == '(' {
			open++
		}
		if r == ')' {
			open--
		}
		if open == 0 {
			break
		}
		end++
	}

	if open != 0 {
		return Subject{}, false, rs[start:]
	}

	/* depleted the input stream early, was expecting a ++ or --
	 * at the end
	 */
	if len(rs) == end {
		return Subject{}, true, rs[end:]
	}

	name := string(rs[start:end])

	karmaStart := end + 1       // index of the first + or -
	karmaEnd := karmaStart + 1  // index of the second + or -
	expectSpace := karmaEnd + 1 // index where whitespace is expected or eof

	/* a valid subject will have whitespace after the ++/-- OR
	 * the ++/-- is the end of the input stream
	 */
	sepBySpace := len(rs) > expectSpace && unicode.IsSpace(rs[expectSpace])
	eof := len(rs)-1 == karmaEnd

	if sepBySpace || eof {
		if rs[karmaStart] == rs[karmaEnd] {
			var karma int
			switch rs[karmaStart] {
			case '+':
				karma = 1
			case '-':
				karma = -1
			}

			sub := Subject{}
			if len(name) > 0 && karma != 0 {
				sub.Name = name
				sub.Karma = karma
			}
			return sub, true, rs[karmaEnd:]
		}
	}

	return Subject{}, true, rs[end:]
}

func tryParsePlain(rs []rune) (Subject, bool, []rune) {
	end := 0
	for _, r := range rs {
		if unicode.IsSpace(r) || r == '`' {
			break
		}
		end++
	}

	var sub Subject
	raw := string(rs[:end])
	separated := (end < len(rs) && unicode.IsSpace(rs[end])) || len(rs) == end

	/* length check is to avoid parsing "++" as {"", 1} */
	if separated && len(raw) > 2 {
		if strings.HasSuffix(raw, "++") {
			sub.Karma = 1
		} else if strings.HasSuffix(raw, "--") {
			sub.Karma = -1
		}
		if sub.Karma != 0 {
			sub.Name = raw[:len(raw)-2]
		}
		return sub, true, rs[end:]
	}
	sub.Name = raw
	return sub, true, rs[end:]
}
