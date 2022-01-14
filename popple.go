package popple

import (
	"bufio"
	"database/sql"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/connorkuehl/popple/adapter"
	"github.com/connorkuehl/popple/internal/karma"
	"github.com/connorkuehl/popple/internal/popple"
)

var (
	ErrInvalidLimit           = errors.New("limit must be positive non-zero integer")
	ErrInvalidAnnounceSetting = errors.New("announce setting is invalid")
	ErrMissingArgument        = errors.New("expected argument")
)

var (
	defaultLeaderboardSize = 10
)

type Popple struct {
	pl adapter.PersistenceLayer
}

func New(pl adapter.PersistenceLayer) *Popple {
	p := Popple{
		pl: pl,
	}

	return &p
}

func (p *Popple) BumpKarma(serverID string, body io.Reader) (map[string]int, bool, error) {
	cfgf := popple.GetConfig(p.pl, serverID)

	var text strings.Builder
	_, err := io.Copy(&text, body)
	if err != nil {
		return nil, false, err
	}

	bumps := karma.Parse(text.String())

	levels := make(map[string]int)
	for n, k := range bumps {
		if k == 0 {
			continue
		}

		updated, err := p.pl.AddKarmaToEntity(
			adapter.Entity{
				Name:     n,
				ServerID: serverID,
			},
			k,
		)
		if err != nil {
			return nil, false, err
		}

		levels[n] = int(updated.Karma)
	}

	cfgr := <-cfgf
	cfg, err := cfgr.C, cfgr.Err
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, false, err
		}
		err = nil
	}

	return levels, cfg.NoAnnounce, err
}

func (p *Popple) SetAnnounce(serverID string, body io.Reader) error {
	scanner := bufio.NewScanner(body)
	scanner.Split(bufio.ScanWords)
	if ok := scanner.Scan(); !ok {
		err := scanner.Err()
		if err == nil {
			return ErrMissingArgument
		}
		return err
	}

	setting := scanner.Text()

	var on bool
	switch setting {
	case "on", "yes":
		on = true
	case "off", "no":
		on = false
	default:
		return ErrInvalidAnnounceSetting
	}

	_ = p.pl.CreateConfig(serverID)
	return p.pl.PutConfig(adapter.Config{ServerID: serverID, NoAnnounce: !on})
}

func (p *Popple) Karma(serverID string, body io.Reader) (map[string]int, error) {
	var text strings.Builder

	_, err := io.Copy(&text, body)
	if err != nil {
		return nil, err
	}

	who := karma.Parse(text.String())

	levelsr := <-popple.GetLevels(p.pl, serverID, who)
	levels, err := levelsr.Levels, levelsr.Err
	if err != nil {
		return nil, err
	}

	return levels, nil
}

func (p *Popple) Leaderboard(serverID string, top bool, body io.Reader) ([]adapter.LeaderboardEntry, error) {
	limit := defaultLeaderboardSize

	scanner := bufio.NewScanner(body)
	scanner.Split(bufio.ScanWords)
	if ok := scanner.Scan(); !ok {
		err := scanner.Err()
		if err != nil {
			return nil, err
		}
	} else {
		parsedLimit, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return nil, err
		}
		if parsedLimit < 1 {
			return nil, ErrInvalidLimit
		}
		limit = parsedLimit
	}

	lbr := <-popple.GetLeaderboard(p.pl, serverID, top, uint(limit))
	entries, err := lbr.Entries, lbr.Err
	if err != nil {
		return nil, err
	}

	return entries, nil
}
