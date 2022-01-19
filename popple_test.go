package popple

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/connorkuehl/popple/adapter"
)

func assertPersisted(t *testing.T, pl adapter.PersistenceLayer, serverID string, want map[string]int) {
	for n, k := range want {
		ent, err := pl.GetEntity(serverID, n)
		if k == 0 {
			if !errors.Is(err, sql.ErrNoRows) {
				t.Errorf("got %v, want %v", err, sql.ErrNoRows)
			}
			return
		}

		if int(ent.Karma) != k {
			t.Errorf("got karma=%d, want karma=%d", ent.Karma, k)
			return
		}
	}
}

func withEntities(t *testing.T, entities ...adapter.Entity) func(pl adapter.PersistenceLayer) {
	return func(pl adapter.PersistenceLayer) {
		for _, entity := range entities {
			_ = pl.CreateEntity(entity.ServerID, entity.Name)
			_, err := pl.AddKarmaToEntity(entity, int(entity.Karma))
			if err != nil {
				t.Errorf("failed to add entity %+v to test db: %v", entity, err)
				return
			}
		}
	}
}

func newTestDB(t *testing.T, opts ...func(pl adapter.PersistenceLayer)) (adapter.PersistenceLayer, func()) {
	var cleanup func()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Errorf("failed to open in-memory sql database: %v", err)
		return nil, func() {}
	}
	cleanup = func() {
		db.Close()
	}

	pl, err := adapter.NewSQLitePersistenceLayer(db)
	if err != nil {
		t.Errorf("failed to make tables for in-memory database: %v", err)
		return nil, func() {}
	}

	for _, opt := range opts {
		opt(pl)
	}

	return pl, cleanup
}

func TestBumpKarma(t *testing.T) {
	t.Run("no karma bumps", func(t *testing.T) {
		pl, cleanup := newTestDB(t)
		defer cleanup()

		p := New(pl)
		levels, _, err := p.BumpKarma("1", strings.NewReader("hello, world"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if len(levels) != 0 {
			t.Errorf("unexpected karma bumps")
			return
		}
	})

	bumps := []struct {
		serverID string
		body     string
		want     map[string]int
	}{
		{"1", "a++ b-- c++ c++", map[string]int{"a": 1, "b": -1, "c": 2}},
		{"2", "(a b c d)++", map[string]int{"a b c d": 1}},
		{"3", "((nest))--", map[string]int{"(nest)": -1}},
		{"4", "a++ `b--", map[string]int{"a": 1}},
	}

	for _, tt := range bumps {
		t.Run(tt.body, func(t *testing.T) {
			pl, cleanup := newTestDB(t)
			defer cleanup()

			p := New(pl)
			levels, _, err := p.BumpKarma(tt.serverID, strings.NewReader(tt.body))
			if err != nil {
				t.Errorf("unexpected err: %v", err)
				return
			}

			if !reflect.DeepEqual(levels, tt.want) {
				t.Errorf("got %v, want %v", levels, tt.want)
				return
			}

			assertPersisted(t, pl, tt.serverID, tt.want)
		})
	}

	// seed the persistence layer for the preexisting test cases below.
	preexistingDBFixture := func(t *testing.T) (adapter.PersistenceLayer, func()) {
		entities := []adapter.Entity{
			{ServerID: "5", Name: "a", Karma: 1},
			{ServerID: "5", Name: "in parens", Karma: -1},
		}

		pl, cleanup := newTestDB(t, withEntities(t, entities...))
		return pl, cleanup
	}

	// note: these assertions are based on the preexistingDBFixture that is
	// freshly created for each test case.
	preexisting := []struct {
		serverID string
		body     string
		want     map[string]int
	}{
		{"5", "a++", map[string]int{"a": 2}},
		{"5", "(in parens)++", map[string]int{"in parens": 0}},
		{"5", "b--", map[string]int{"b": -1}},
	}

	for _, tt := range preexisting {
		t.Run(tt.body, func(t *testing.T) {
			pl, cleanup := preexistingDBFixture(t)
			defer cleanup()

			p := New(pl)
			levels, _, err := p.BumpKarma(tt.serverID, strings.NewReader(tt.body))
			if err != nil {
				t.Errorf("unexpected err: %v", err)
				return
			}
			if !reflect.DeepEqual(levels, tt.want) {
				t.Errorf("got %v, want %v", levels, tt.want)
				return
			}

			assertPersisted(t, pl, tt.serverID, tt.want)
		})
	}
}

func TestSetAnnounce(t *testing.T) {
	t.Run("missing argument", func(t *testing.T) {
		pl, cleanup := newTestDB(t)
		defer cleanup()

		p := New(pl)
		err := p.SetAnnounce("100", strings.NewReader(""))
		if err == nil {
			t.Errorf("expected an err, didn't get one")
			return
		}
		if !errors.Is(err, ErrMissingArgument) {
			t.Errorf("got %v, want %v", err, ErrMissingArgument)
			return
		}
	})

	t.Run("invalid setting", func(t *testing.T) {
		pl, cleanup := newTestDB(t)
		defer cleanup()

		p := New(pl)
		err := p.SetAnnounce("101", strings.NewReader("hjkl"))
		if err == nil {
			t.Errorf("expected an err, didn't get one")
			return
		}
		if !errors.Is(err, ErrInvalidAnnounceSetting) {
			t.Errorf("got %v, want %v", err, ErrInvalidAnnounceSetting)
			return
		}
	})

	assertSetAnnounce := func(t *testing.T, serverID, cmd string, want bool) {
		pl, cleanup := newTestDB(t)
		defer cleanup()

		p := New(pl)
		err := p.SetAnnounce(serverID, strings.NewReader(cmd))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		cfg, err := pl.GetConfig(serverID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if cfg.NoAnnounce != want {
			t.Errorf("got NoAnnounce=%v, want %v", cfg.NoAnnounce, want)
			return
		}
	}

	cases := []struct {
		serverID string
		cmd      string
		want     bool
	}{
		{"201", "on", false},
		{"202", "yes", false},
		{"203", "off", true},
		{"204", "no", true},
	}

	for _, tt := range cases {
		t.Run(tt.cmd, func(t *testing.T) {
			assertSetAnnounce(t, tt.serverID, tt.cmd, tt.want)
		})
	}
}

func TestCheckKarma(t *testing.T) {
	t.Run("no karma", func(t *testing.T) {
		pl, cleanup := newTestDB(t)
		defer cleanup()

		p := New(pl)

		levels, err := p.Karma("8", strings.NewReader("hotel echo lima little oscar"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		want := map[string]int{
			"hotel":  0,
			"echo":   0,
			"lima":   0,
			"little": 0,
			"oscar":  0,
		}

		if !reflect.DeepEqual(levels, want) {
			t.Errorf("got %v, want %v", levels, want)
		}
	})

	dbFixture := func(t *testing.T) (adapter.PersistenceLayer, func()) {
		entities := []adapter.Entity{
			{ServerID: "26", Name: "z", Karma: 127},
			{ServerID: "26", Name: "in parens", Karma: -127},
		}

		pl, cleanup := newTestDB(t, withEntities(t, entities...))
		return pl, cleanup
	}

	t.Run("varying karma", func(t *testing.T) {
		pl, cleanup := dbFixture(t)
		defer cleanup()

		p := New(pl)

		levels, err := p.Karma("26", strings.NewReader("z (in parens) not_here"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		want := map[string]int{
			"z":         127,
			"in parens": -127,
			"not_here":  0,
		}

		if !reflect.DeepEqual(levels, want) {
			t.Errorf("got %v, want %v", levels, want)
			return
		}
	})
}

func TestLeaderboard(t *testing.T) {
	dbFixture := func(t *testing.T) (adapter.PersistenceLayer, func()) {
		entities := []adapter.Entity{
			{ServerID: "66", Name: "g", Karma: 7},
			{ServerID: "66", Name: "a", Karma: 1},
			{ServerID: "66", Name: "b", Karma: 2},
			{ServerID: "66", Name: "j", Karma: 10},
			{ServerID: "66", Name: "d", Karma: 4},
			{ServerID: "66", Name: "f", Karma: 6},
			{ServerID: "66", Name: "c", Karma: 3},
			{ServerID: "66", Name: "h", Karma: 8},
			{ServerID: "66", Name: "i", Karma: 9},
			{ServerID: "66", Name: "e", Karma: 5},
		}

		pl, cleanup := newTestDB(t, withEntities(t, entities...))
		return pl, cleanup
	}

	t.Run("no argument uses the default leaderboard size", func(t *testing.T) {
		testLeaderboardSize := 3
		dls := defaultLeaderboardSize
		defaultLeaderboardSize = testLeaderboardSize
		defer func() { defaultLeaderboardSize = dls }()

		pl, cleanup := dbFixture(t)
		defer cleanup()

		p := New(pl)

		entries, err := p.Leaderboard("66", true, strings.NewReader(""))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if len(entries) != testLeaderboardSize {
			t.Errorf("got %d, want %d", len(entries), testLeaderboardSize)
			return
		}
	})

	for i := -1; i < 1; i++ {
		t.Run(fmt.Sprintf("leaderboard with bad limit=%d", i), func(t *testing.T) {
			pl, cleanup := newTestDB(t)
			defer cleanup()
			p := New(pl)

			_, err := p.Leaderboard("1234", true, strings.NewReader(fmt.Sprintf("%d", i)))
			if err == nil {
				t.Errorf("expected error, but didn't get one")
				return
			}
			if !errors.Is(err, ErrInvalidLimit) {
				t.Errorf("got %v, want %v", err, ErrInvalidLimit)
				return
			}
		})
	}

	t.Run("valid limit", func(t *testing.T) {
		pl, cleanup := newTestDB(t)
		defer cleanup()
		p := New(pl)

		_, err := p.Leaderboard("4321", true, strings.NewReader("1"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
	})

	t.Run("leaderboard is sorted from highest karma to lowest", func(t *testing.T) {
		entities := []adapter.Entity{
			{ServerID: "99", Name: "c", Karma: 8},
			{ServerID: "99", Name: "b", Karma: 9},
			{ServerID: "99", Name: "a", Karma: 10},
		}
		pl, cleanup := newTestDB(t, withEntities(t, entities...))
		defer cleanup()

		p := New(pl)

		entries, err := p.Leaderboard("99", true, strings.NewReader(""))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		want := []adapter.LeaderboardEntry{
			{Name: "a", Karma: 10},
			{Name: "b", Karma: 9},
			{Name: "c", Karma: 8},
		}

		if !reflect.DeepEqual(entries, want) {
			t.Errorf("got %v, want %v", entries, want)
			return
		}
	})

	t.Run("loserboard is sorted from lowest karma to highest", func(t *testing.T) {
		entities := []adapter.Entity{
			{ServerID: "77", Name: "a", Karma: 10},
			{ServerID: "77", Name: "b", Karma: 9},
			{ServerID: "77", Name: "c", Karma: 8},
		}
		pl, cleanup := newTestDB(t, withEntities(t, entities...))
		defer cleanup()

		p := New(pl)

		entries, err := p.Leaderboard("77", false, strings.NewReader(""))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		want := []adapter.LeaderboardEntry{
			{Name: "c", Karma: 8},
			{Name: "b", Karma: 9},
			{Name: "a", Karma: 10},
		}

		if !reflect.DeepEqual(entries, want) {
			t.Errorf("got %v, want %v", entries, want)
			return
		}
	})
}
