package popple

import (
	"bufio"
	"strings"
	"testing"
)

func TestMux(t *testing.T) {
	t.Run("command prefix is stripped from message", func(t *testing.T) {
		mux := NewMux("@colt")
		_, got := mux.Route("@colt announce on")

		wordscanner := bufio.NewScanner(strings.NewReader(got))
		wordscanner.Split(bufio.ScanWords)

		wordscanner.Scan()
		got = wordscanner.Text()

		want := "on"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("announce route", func(t *testing.T) {
		m := NewMux("bull")
		got, _ := m.Route("bull announce")
		_ = got.(AnnounceHandler)
	})

	t.Run("karma route", func(t *testing.T) {
		m := NewMux("goat")
		got, _ := m.Route("goat karma")
		_ = got.(KarmaHandler)
	})

	t.Run("leaderboard route", func(t *testing.T) {
		m := NewMux("hare")
		got, _ := m.Route("hare top")
		_ = got.(LeaderboardHandler)
	})

	t.Run("loserboard route", func(t *testing.T) {
		m := NewMux("mule")
		got, _ := m.Route("mule bot")
		_ = got.(LoserboardHandler)
	})

	t.Run("the default route is bump karma", func(t *testing.T) {
		m := NewMux("newt")
		got, _ := m.Route("wolf seal oxen")
		_ = got.(BumpKarmaHandler)
	})

	t.Run("it allows any amount of whitespace between the name and cmd", func(t *testing.T) {
		t.Skip("https://github.com/connorkuehl/popple/issues/112")

		tests := []struct {
			in    string
			check func(got interface{})
		}{
			{
				in: "oryx  announce",
				check: func(got interface{}) {
					_ = got.(AnnounceHandler)
				},
			},
			{
				in: "oryx       karma",
				check: func(got interface{}) {
					_ = got.(KarmaHandler)
				},
			},
			{
				in: "oryx             top",
				check: func(got interface{}) {
					_ = got.(LeaderboardHandler)
				},
			},
			{
				in: "oryx     bot",
				check: func(got interface{}) {
					_ = got.(LoserboardHandler)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.in, func(t *testing.T) {
				m := NewMux("oryx")

				got, _ := m.Route(tt.in)
				tt.check(got)
			})
		}
	})

	t.Run("it requires a space before arguments to commands", func(t *testing.T) {
		t.Skip("https://github.com/connorkuehl/popple/issues/123")

		tests := []struct {
			in    string
			check func(got interface{})
			want  string
		}{
			{
				in:    "deer announce on",
				check: func(got interface{}) { _ = got.(AnnounceHandler) },
				want:  "on",
			},
			{
				in:    "deer karma  person1",
				check: func(got interface{}) { _ = got.(KarmaHandler) },
				want:  "person1",
			},
			{
				in:    "deer top 		2",
				check: func(got interface{}) { _ = got.(LeaderboardHandler) },
				want:  "2",
			},
			{
				in:    "deer bot 12",
				check: func(got interface{}) { _ = got.(LoserboardHandler) },
				want:  "12",
			},
			{
				in:    "deer announceon",
				check: func(got interface{}) { _ = got.(BumpKarmaHandler) },
				want:  "deer announceon",
			},
		}

		for _, tt := range tests {
			t.Run(tt.in, func(t *testing.T) {
				m := NewMux("deer")
				hnd, rem := m.Route(tt.in)
				tt.check(hnd)

				if rem != tt.want {
					t.Errorf("want %q, got %q", tt.want, rem)
				}
			})
		}
	})
}
