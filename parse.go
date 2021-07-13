package main

import (
	"strings"
	"unicode"
)

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

// Subject represents a karma operation on a named entity.
type Subject struct {
	Name  string
	Karma int
}

// ParseSubjects parses subjects from text.
func ParseSubjects(s string) []Subject {
	subjects := make([]Subject, 0)

	_, items := lex([]rune(s))
	for {
		if i, ok := <-items; ok {
			s := parseSubject(i)
			if len(s.Name) == 0 {
				continue
			}
			subjects = append(subjects, s)
		} else {
			break
		}
	}

	return subjects
}

func lex(input []rune) (*lexer, chan item) {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run()

	return l, l.items
}

func parseSubject(i item) Subject {
	switch i.kind {
	case itemText:
		return parseSubjectPlain(i)
	case itemTextInParens:
		return parseSubjectParens(i)
	default:
		return Subject{}
	}
}

func parseSubjectPlain(i item) Subject {
	name := string(i.value)
	name = strings.TrimPrefix(name, "@")

	karma := 0
	switch {
	case strings.HasSuffix(name, "++"):
		karma = 1
	case strings.HasSuffix(name, "--"):
		karma = -1
	}
	if karma != 0 {
		name = name[:len(name)-2]
	}
	return Subject{name, karma}
}

func parseSubjectParens(i item) Subject {
	name := string(i.value)
	karma := 0
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
	return Subject{name, karma}
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
		l.start = l.pos
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
		return discardTick
	case unicode.IsSpace(ch):
		return discardSpace
	default:
		return lexText
	}
}

func lexText(l *lexer) stateFn {
	for ch := l.next(); ch != eof; {
		if unicode.IsSpace(ch) {
			l.backup()
			break
		}
		if ch == tick {
			break
		}
		ch = l.next()
	}

	l.emit(itemText)

	return lexEntry
}

func discardTick(l *lexer) stateFn {
	l.next()
	l.ignore()
	restart := l.first() // make note of where we are

	for ch := l.next(); ch != eof; {
		if ch == tick {
			l.ignore()
			return lexEntry
		}
		ch = l.next()
	}

	// didn't find a matching tick
	if l.peek() == eof {
		l.set(restart)
		if l.peek() == eof {
			return nil
		}
		return lexEntry
	}

	// discard all the way up to this tick
	l.ignore()

	return lexEntry
}

func discardSpace(l *lexer) stateFn {
	for ch := l.next(); ch != eof; {
		if !unicode.IsSpace(ch) {
			l.backup()
			break
		}
		ch = l.next()
	}

	ch := l.peek()
	if ch == eof {
		return nil
	}
	l.ignore()

	return lexEntry
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
