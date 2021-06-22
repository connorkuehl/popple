package main

// Module parse (unsurprisingly) parses karma subjects from text.
//
// A karma subject is an entity whose karma is either incremented
// or decremented.
//
// A valid subject could be a word containing any kinds of characters
// (except a backtick `) ending in either a `++`` or a `--``.
//
// Example: HelloWorld++ -> subject name: HelloWorld +1 karma
// Example: Good`Bye++ -> subject name: Bye +1 karma (note that
// the backtick excluded the first part of the word from the subject)
//
// Subjects can contain whitespace or any other characters if they are
// enclosed in parentheses.
//
// Example: (Hello World)-- -> subject name: Hello World -1 karma

import (
	"strings"
	"unicode"
)

// Subject represents a karma operation on a named entity.
type Subject struct {
	Name  string
	Karma int
}

// ParseSubjects parses subjects from text.
func ParseSubjects(s string) []Subject {
	subjects := make([]Subject, 0)
	var sub Subject
	var ok bool
	var remaining []rune

	remaining = []rune(s)
	for len(remaining) > 0 {
		sub, ok, remaining = tryParseSubject(remaining)
		if !ok || len(sub.Name) == 0 {
			continue
		}
		subjects = append(subjects, sub)
	}
	return subjects
}

// tryParseSubject is the general entrypoint for parsing subjects
// from text.
//
// This function, and all of the functions that it calls, work by
// taking as input the text "input stream". Regardless of whether
// it is successful (i.e., it returns a Subject object and the
// returned 'ok' bool is true), it will also return the *remaining*
// input stream that it did not consume to produce the subject.
//
// This mechanism is helpful as it allows callers to loop over this
// function as if it were an iterator. The other benefit to this
// "consume-and-return-the-remainder" approach is that it allows
// the parser to "backtrack" in case it consumed the entire input
// looking for a matching backtick or closing parenthesis.
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
		if len(remaining) == 0 && !ok {
			return Subject{}, false, rs[1:]
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

// seekSubjectStart discards any irrelevant items in the
// input stream so that the input stream points to a possible
// karma subject.
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

// tryParseParens is called when the first character in the
// input stream is an opening parenthesis. It will read
// everything until the parenthesis is closed. Once closed,
// it will attempt to read the karma increment or decrement
// following the closing parenthesis.
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

	sub := Subject{string(rs[start:end]), 0}

	/* depleted the input stream early, was expecting a ++ or --
	 * at the end
	 */
	if len(rs)-1 == end {
		return sub, true, []rune{}
	}

	karmaStart := end + 1       // index of the first + or -
	karmaEnd := karmaStart + 1  // index of the second + or -
	expectSpace := karmaEnd + 1 // index where whitespace is expected or eof

	/* a valid subject will have whitespace after the ++/-- OR
	 * the ++/-- is the end of the input stream
	 */
	sepBySpace := len(rs) > expectSpace && unicode.IsSpace(rs[expectSpace])
	eof := len(rs)-1 == karmaEnd

	if sepBySpace || eof {
		if rs[karmaStart] == '+' && rs[karmaEnd] == '+' {
			sub.Karma = 1
		} else if rs[karmaStart] == '-' && rs[karmaEnd] == '-' {
			sub.Karma = -1
		}
		return sub, true, rs[karmaEnd+1:]
	}

	return sub, true, rs[end+1:]
}

// tryParsePlain is a catch-all parser. It will essentially read
// until it encounters whitespace and then check if the last two
// characters are a karma operation.
func tryParsePlain(rs []rune) (Subject, bool, []rune) {
	end := 0
	for _, r := range rs {
		if unicode.IsSpace(r) || r == '`' {
			break
		}
		end++
	}

	sub := Subject{string(rs[:end]), 0}
	separated := (end < len(rs) && unicode.IsSpace(rs[end])) || len(rs) == end
	if separated {
		switch {
		case strings.HasSuffix(sub.Name, "++"):
			sub.Karma = 1
		case strings.HasSuffix(sub.Name, "--"):
			sub.Karma = -1
		}
		if sub.Karma != 0 {
			sub.Name = sub.Name[:len(sub.Name)-2]
		}
	}
	return sub, true, rs[end:]
}
