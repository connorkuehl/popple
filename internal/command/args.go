package command

import (
	"bufio"
	"errors"
	"strconv"
	"strings"

	"github.com/connorkuehl/popple/internal/popple"
)

var (
	ErrMissingArgument = errors.New("missing argument")
	ErrInvalidArgument = errors.New("invalid argument")
)

var DefaultLimit uint = 10

type SetAnnounceArgs struct {
	NoAnnounce bool
}

func (args *SetAnnounceArgs) ParseArg(s string) error {
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Split(bufio.ScanWords)
	if ok := scanner.Scan(); !ok {
		err := scanner.Err()
		if err == nil {
			return ErrMissingArgument
		}
		return err
	}

	var on bool
	setting := scanner.Text()
	switch setting {
	case "on", "yes":
		on = true
	case "off", "no":
		on = false
	default:
		return ErrInvalidArgument
	}

	args.NoAnnounce = !on
	return nil
}

type ChangeKarmaArgs struct {
	Increments popple.Increments
}

func (args *ChangeKarmaArgs) ParseArg(s string) error {
	args.Increments = popple.ParseIncrements(s)

	filtered := make(popple.Increments)
	for who, inc := range args.Increments {
		if inc == 0 {
			continue
		}
		filtered[who] = inc
	}

	args.Increments = filtered
	return nil
}

type CheckKarmaArgs struct {
	Who []string
}

func (args *CheckKarmaArgs) ParseArg(s string) error {
	var who []string

	increments := popple.ParseIncrements(s)
	for name := range increments {
		who = append(who, name)
	}

	args.Who = who
	return nil
}

type LeaderboardArgs struct {
	BoardArgs
}

func (args *LeaderboardArgs) ParseArg(s string) error {
	args.Order = popple.BoardOrderDsc
	return args.BoardArgs.ParseArg(s)
}

type LoserboardArgs struct {
	BoardArgs
}

func (args *LoserboardArgs) ParseArg(s string) error {
	args.Order = popple.BoardOrderAsc
	return args.BoardArgs.ParseArg(s)
}

type BoardArgs struct {
	Limit uint
	Order popple.BoardOrder
}

func (args *BoardArgs) ParseArg(s string) error {
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Split(bufio.ScanWords)

	if ok := scanner.Scan(); !ok {
		err := scanner.Err()
		if err != nil {
			return err
		}
		args.Limit = DefaultLimit
	} else {
		parsedLimit, err := strconv.Atoi(scanner.Text())
		if err != nil {
			return ErrInvalidArgument
		}
		if parsedLimit < 1 {
			return ErrInvalidArgument
		}
		args.Limit = uint(parsedLimit)
	}

	return nil
}
