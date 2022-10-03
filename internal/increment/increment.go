package increment

import (
	"strings"
	"unicode"
)

func ParseAll(s string) map[string]int64 {
	increments := make(map[string]int64)
	_, items := lex([]rune(s))
	for i := range items {
		name, incr := parseIncrement(i)
		if len(name) == 0 {
			continue
		}
		increments[name] += incr
	}
	return increments
}

func lex(input []rune) (*lexer, chan item) {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run()

	return l, l.items
}

func parseIncrement(i item) (name string, increment int64) {
	switch i.kind {
	case itemText:
		return parseIncrementPlain(i)
	case itemTextInParens:
		return parseIncrementInParens(i)
	default:
		return "", 0
	}
}

func parseIncrementPlain(i item) (name string, increment int64) {
	name = string(i.value)
	// TODO: get rid of this
	name = strings.TrimPrefix(name, "@")

	karma := int64(0)
	switch {
	case strings.HasSuffix(name, "++"):
		karma = 1
	case strings.HasSuffix(name, "--"):
		karma = -1
	}
	if karma != 0 {
		name = name[:len(name)-2]
	}
	return name, karma
}

func parseIncrementInParens(i item) (name string, increment int64) {
	name = string(i.value)
	karma := int64(0)
	switch {
	case strings.HasSuffix(name, ")++"):
		karma = 1
	case strings.HasSuffix(name, ")--"):
		karma = -1
	}
	switch {
	case karma != 0:
		name = name[1 : len(name)-len(")..")]
	default:
		name = name[1 : len(name)-1]
	}
	return name, karma
}

type lexer struct {
	input []rune
	start int
	pos   int
	items chan item
}

func (l *lexer) run() {
	for state := lexEntry; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func (l *lexer) emit(t itemType) {
	if l.pos > l.start {
		l.items <- item{t, l.input[l.start:l.pos]}
		l.ignore()
	}
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		return eof
	}
	hold := l.pos
	l.pos++
	return l.input[hold]
}

func (l *lexer) backup() {
	if l.pos > 0 {
		l.pos--
	}
}

func (l *lexer) peek() rune {
	r := l.next()
	if r != eof {
		l.backup()
	}
	return r
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) first() int {
	return l.start
}

func (l *lexer) set(idx int) {
	if idx < 0 || idx >= len(l.input) {
		return
	}
	l.start = idx
	l.pos = idx
}

func (l *lexer) accept(rs []rune) bool {
	if inRunes(rs, l.next()) {
		return true
	}
	l.backup()
	return false
}

func inRunes(rs []rune, r rune) bool {
	for _, rune := range rs {
		if rune == r {
			return true
		}
	}
	return false
}

const eof rune = 0

type item struct {
	kind  itemType
	value []rune
}

type itemType int

const (
	itemText         itemType = iota // alphanumeric
	itemTextInParens                 // (alphanumeric)
)

const tick rune = '`'
const openParen rune = '('
const closedParen rune = ')'

type stateFn func(*lexer) stateFn

func lexEntry(l *lexer) stateFn {
	ch := l.peek()
	switch {
	case ch == eof:
		return nil
	case ch == openParen:
		return lexInParen
	case ch == tick:
		return nil
	case unicode.IsSpace(ch):
		return discardSpace
	default:
		return lexText
	}
}

func lexText(l *lexer) stateFn {
	for {
		ch := l.next()
		if unicode.IsSpace(ch) {
			l.backup()
			break
		} else if ch == tick {
			l.backup()
			return discardTick
		} else if ch == eof {
			break
		}
	}

	l.emit(itemText)

	return lexEntry
}

func discardTick(l *lexer) stateFn {
	l.ignore()
	return nil
}

func discardSpace(l *lexer) stateFn {
	for {
		ch := l.next()
		if ch == eof {
			return nil
		}
		if !unicode.IsSpace(ch) {
			l.backup()
			l.ignore()
			return lexEntry
		}
	}
}

func lexInParen(l *lexer) stateFn {
	restart := l.first()

	open := 0
	for ch := l.next(); ch != eof; {
		switch ch {
		case openParen:
			open++
		case closedParen:
			open--
		}
		if open == 0 {
			break
		}
		ch = l.next()
	}

	// didn't close the paren
	if l.peek() == eof && open != 0 {
		l.set(restart + 1)
		return lexEntry
	}

	// no karma operation trailing after this
	if l.peek() == eof || unicode.IsSpace(l.peek()) {
		l.emit(itemTextInParens)
		return lexEntry
	}

	acceptable := []rune("+-")

	// + or -
	l.accept(acceptable)
	if unicode.IsSpace(l.peek()) || l.peek() == eof {
		l.backup()
		return lexEntry
	}
	l.accept(acceptable)
	end := l.next()
	if unicode.IsSpace(end) {
		l.backup()
	}

	if unicode.IsSpace(end) || end == eof {
		l.emit(itemTextInParens)
	}

	return lexEntry
}
