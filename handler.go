package popple

import (
	"bufio"
	"strconv"
	"strings"

	poperr "github.com/connorkuehl/popple/errors"
	"github.com/connorkuehl/popple/get"
	"github.com/connorkuehl/popple/parse"
)

type AnnounceHandler func(repo Repository, serverID string, on bool) error

func ParseAnnounceArgs(message string) (on bool, err error) {
	scanner := bufio.NewScanner(strings.NewReader(message))
	scanner.Split(bufio.ScanWords)
	if ok := scanner.Scan(); !ok {
		err := scanner.Err()
		if err == nil {
			return false, poperr.ErrMissingArgument
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
		return false, poperr.ErrInvalidArgument
	}

	return on, nil
}

type BumpKarmaHandler func(repo Repository, serverID string, increments map[string]int64) (levels map[string]int64, err error)

func ParseBumpKarmaArgs(message string) (increments map[string]int64, err error) {
	increments = parse.Subjects(message)
	return increments, nil
}

type KarmaHandler func(repo get.EntityRepository, serverID string, who map[string]struct{}) (levels map[string]int64, err error)

func ParseKarmaArgs(message string) (who map[string]struct{}, err error) {
	increments := parse.Subjects(message)
	if len(increments) < 1 {
		return nil, poperr.ErrMissingArgument
	}
	who = make(map[string]struct{})
	for name := range increments {
		who[name] = struct{}{}
	}
	return who, nil
}

type LeaderboardHandler func(repo get.EntityRepository, serverID string, limit uint) ([]get.Entity, error)

func ParseLeaderboardArgs(message string) (limit uint, err error) {
	return ParseBoardArgs(message)
}

type LoserboardHandler func(repo get.EntityRepository, serverID string, limit uint) ([]get.Entity, error)

func ParseLoserboardArgs(message string) (limit uint, err error) {
	return ParseBoardArgs(message)
}

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
			return 0, poperr.ErrInvalidArgument
		}
		if parsedLimit < 1 {
			return 0, poperr.ErrInvalidArgument
		}
		limit = uint(parsedLimit)
	}

	return limit, nil
}
