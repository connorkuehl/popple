package popple

import (
	"bufio"
	"strconv"
	"strings"

	"github.com/connorkuehl/popple/internal/parse"
)

// AnnounceHandler is the type returned by the Mux to denote that the action
// being taken should be forwarded to Announce.
type AnnounceHandler func(repo Repository, serverID string, on bool) error

// ParseAnnounceArgs parses the arguments that must be forwarded to the
// AnnounceHandler.
func ParseAnnounceArgs(message string) (on bool, err error) {
	scanner := bufio.NewScanner(strings.NewReader(message))
	scanner.Split(bufio.ScanWords)
	if ok := scanner.Scan(); !ok {
		err := scanner.Err()
		if err == nil {
			return false, ErrMissingArgument
		}
		return false, err
	}

	setting := scanner.Text()
	switch setting {
	case "on", "yes":
		on = true
	case "off", "no":
		on = false
	default:
		return false, ErrInvalidArgument
	}

	return on, nil
}

// BumpKarmaHandler is the type returned by the Mux to denote that the action
// being taken should be forwarded to BumpKarma.
type BumpKarmaHandler func(repo Repository, serverID string, increments map[string]int64) (levels map[string]int64, err error)

// ParseBumpKarmaArgs
func ParseBumpKarmaArgs(message string) (increments map[string]int64, err error) {
	increments = parse.Subjects(message)
	return increments, nil
}

// KarmaHandler is the type returned by the Mux to denote that the action being
// taken should be forwarded to Karma.
type KarmaHandler func(repo Repository, serverID string, who map[string]struct{}) (levels map[string]int64, err error)

// ParseKarmaArgs parses the arguments that must be forwarded to the
// KarmaHandler.
func ParseKarmaArgs(message string) (who map[string]struct{}, err error) {
	increments := parse.Subjects(message)
	if len(increments) < 1 {
		return nil, ErrMissingArgument
	}
	who = make(map[string]struct{})
	for name := range increments {
		who[name] = struct{}{}
	}
	return who, nil
}

// LeaderboardHandler is the type returned by the Mux to denote that the action being
// taken should be forwarded to Leaderboard.
type LeaderboardHandler func(repo Repository, serverID string, limit uint) ([]Entity, error)

// ParseLeaderboardArgs parses the arguments that must be forwarded to the
// LeaderboardHandler.
func ParseLeaderboardArgs(message string) (limit uint, err error) {
	return ParseBoardArgs(message)
}

// LoserboardHandler is the type returned by the Mux to denote that the action being
// taken should be forwarded to Loserboard.
type LoserboardHandler func(repo Repository, serverID string, limit uint) ([]Entity, error)

// ParseLoserboardArgs parses the arguments that must be forwarded to the
// LoserboardHandler.
func ParseLoserboardArgs(message string) (limit uint, err error) {
	return ParseBoardArgs(message)
}

// ParseBoardArgs parses the arguments that must be forwarded to either
// LeaderboardHandler or LoserboardHandler.
//
// TODO: this can be made private
func ParseBoardArgs(message string) (limit uint, err error) {
	scanner := bufio.NewScanner(strings.NewReader(message))
	scanner.Split(bufio.ScanWords)

	if ok := scanner.Scan(); !ok {
		err := scanner.Err()
		if err != nil {
			return 0, err
		}
		limit = defaultLeaderboardSize
	} else {
		parsedLimit, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return 0, ErrInvalidArgument
		}
		if parsedLimit < 1 {
			return 0, ErrInvalidArgument
		}
		limit = uint(parsedLimit)
	}

	return limit, nil
}
