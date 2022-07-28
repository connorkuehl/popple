package popple_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/connorkuehl/popple"
	"github.com/connorkuehl/popple/internal/repo/null"
)

func TestAnnounce(t *testing.T) {
	t.Run("it persists a new config object if one doesn't exist", func(t *testing.T) {
		repo := null.NewRepository()

		err := popple.Announce(repo, "10", false)
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		_, err = repo.Config("10")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	tests := []struct {
		name string
		on   bool
		want bool
	}{
		{
			name: "it sets NoAnnounce to false when on is true",
			on:   true,
			want: false,
		},
		{
			name: "it sets NoAnnounce to true when on is false",
			on:   false,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := null.NewRepository()

			err := popple.Announce(repo, "11", tt.on)
			if err != nil {
				t.Errorf("unexpected err: %v", err)
			}

			config, err := repo.Config("11")
			if err != nil {
				t.Errorf("unexpected err: %v", err)
			}

			if config.NoAnnounce != tt.want {
				t.Errorf("got %v, want %v", config.NoAnnounce, tt.want)
			}

		})
	}
}

func TestBumpKarma(t *testing.T) {
	t.Run("net-zero bumps are no-ops", func(t *testing.T) {
		repo := null.NewRepository()

		got, err := popple.BumpKarma(repo, "1", map[string]int64{"lion": 0})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		want := map[string]int64{}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		_, err = repo.Entity("1", "lion")
		if !errors.Is(err, popple.ErrNotFound) {
			t.Errorf("got %v, want %v", err, popple.ErrNotFound)
		}
	})

	t.Run("bumping a new entity persists a new entity", func(t *testing.T) {
		repo := null.NewRepository()

		_, err := repo.Entity("2", "bear")
		if !errors.Is(err, popple.ErrNotFound) {
			t.Errorf("got %v, want %v", err, popple.ErrNotFound)
		}

		got, err := popple.BumpKarma(repo, "2", map[string]int64{"bear": 1})
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		want := map[string]int64{"bear": 1}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		persisted, err := repo.Entity("2", "bear")
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		if persisted.Name != "bear" || persisted.Karma != 1 {
			t.Errorf("got name=%q karma=%d, want name=%q, karma=%d", persisted.Name, persisted.Karma, "bear", 1)
		}
	})

	t.Run("bumping an existing entity modifies its karma", func(t *testing.T) {
		repo := null.NewRepository()

		err := repo.CreateEntity(popple.Entity{ServerID: "3", Name: "crow"})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		err = repo.UpdateEntity(popple.Entity{ServerID: "3", Name: "crow", Karma: 5})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		got, err := popple.BumpKarma(repo, "3", map[string]int64{"crow": -1})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		want := map[string]int64{"crow": 4}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		persisted, err := repo.Entity("3", "crow")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if persisted.Name != "crow" || persisted.Karma != 4 {
			t.Errorf("got name=%q karma=%d, want name=%q, karma=%d", persisted.Name, persisted.Karma, "crow", 4)
		}
	})

	t.Run("an entity reduced to zero karma is garbage collected", func(t *testing.T) {
		repo := null.NewRepository()

		err := repo.CreateEntity(popple.Entity{ServerID: "4", Name: "orca"})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		err = repo.UpdateEntity(popple.Entity{ServerID: "4", Name: "orca", Karma: -1})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		got, err := popple.BumpKarma(repo, "4", map[string]int64{"orca": 1})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		want := map[string]int64{"orca": 0}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		_, err = repo.Entity("4", "orca")
		if !errors.Is(err, popple.ErrNotFound) {
			t.Errorf("got %v, want %v", err, popple.ErrNotFound)
		}
	})
}

func TestKarma(t *testing.T) {
	t.Run("a non-persisted entity is reported as having zero karma", func(t *testing.T) {
		repo := null.NewRepository()

		got, err := popple.Karma(repo, "5", map[string]struct{}{"lynx": {}})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		want := map[string]int64{"lynx": 0}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("a persisted entity has its persisted karma reported", func(t *testing.T) {
		repo := null.NewRepository()

		err := repo.CreateEntity(popple.Entity{ServerID: "6", Name: "crab"})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		err = repo.UpdateEntity(popple.Entity{ServerID: "6", Name: "crab", Karma: -100})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		got, err := popple.Karma(repo, "6", map[string]struct{}{"crab": {}})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}

		want := map[string]int64{"crab": -100}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

// TODO
func TestLeaderboard(t *testing.T) {
}

// TODO
func TestLoserboard(t *testing.T) {
}
