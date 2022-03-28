// Package popple provides primitives for building a karma-counting bot.
package popple

import (
	"errors"

	"github.com/connorkuehl/popple/create"
	poperr "github.com/connorkuehl/popple/errors"
	"github.com/connorkuehl/popple/get"
	"github.com/connorkuehl/popple/karma"
	"github.com/connorkuehl/popple/remove"
	"github.com/connorkuehl/popple/update"
)

var (
	defaultLeaderboardSize uint = 10
)

type Repository interface {
	create.ConfigRepository
	create.EntityRepository
	get.ConfigRepository
	get.EntityRepository
	remove.EntityRepository
	update.ConfigRepository
	update.EntityRepository
}

// Announce configures a server's "announce" setting.
//
// The announce setting is whether or not the bot should emit the new karma
// levels for whichever entities just had their karma levels bumped.
func Announce(repo Repository, serverID string, on bool) error {
	_, err := repo.Config(serverID)
	if errors.Is(err, poperr.ErrNotFound) {
		err = repo.CreateConfig(create.Config{ServerID: serverID})
		if err != nil {
			return err
		}
	}

	err = repo.UpdateConfig(update.Config{ServerID: serverID, NoAnnounce: !on})

	return err
}

// BumpKarma modifies each entity's karma (the key in the 'increments' map) by
// a relative amount (the value in the 'increments' map).
func BumpKarma(repo Repository, serverID string, increments map[string]int64) (newKarmaLevels map[string]int64, err error) {
	var needsCreate []create.Entity

	// first, collect the current karma levels for the subjects whose karma is being bumped.
	pre := make(map[string]int64)
	for name, incr := range increments {
		// skip net-zero increments, these are no-ops.
		if incr == 0 {
			continue
		}

		entity, err := repo.Entity(serverID, name)
		if errors.Is(err, poperr.ErrNotFound) {
			needsCreate = append(needsCreate, create.Entity{ServerID: serverID, Name: name})
			err = nil
		}
		if err != nil {
			return nil, err
		}

		// this relies on the fact that a zero-value entity is returned when repo.Entity
		// returns an err if the entity is not found.
		pre[name] = entity.Karma
	}

	post := karma.Bump(pre, increments)

	// TODO: consider roll-backs if creating or updating fails.

	for _, create := range needsCreate {
		err = repo.CreateEntity(create)
		if err != nil {
			return nil, err
		}
	}

	for name, karma := range post {
		if karma == 0 {
			// garbage collect the entity, there's no point in storing records with 0 karma.
			err = repo.RemoveEntity(serverID, name)
		} else {
			err = repo.UpdateEntity(update.Entity{ServerID: serverID, Name: name, Karma: karma})
		}

		if err != nil {
			return nil, err
		}
	}

	return post, nil
}

// Karma queries the karma levels for each of the entities (the keys in the 'who'
// map.)
func Karma(repo get.EntityRepository, serverID string, who map[string]struct{}) (levels map[string]int64, err error) {
	levels = make(map[string]int64)
	for name := range who {
		entity, err := repo.Entity(serverID, name)
		if errors.Is(err, poperr.ErrNotFound) {
			err = nil
		}
		if err != nil {
			return nil, err
		}

		// entity should be zero-valued if it happened to be a NotFound err, so if
		// the entity never existed then we'll still report 0 karma for that entity.
		levels[name] = entity.Karma
	}

	return levels, nil
}

// Leaderboard queries the entities with the highest karma levels (the number of
// entities returned will not exceed 'limit.')
func Leaderboard(repo get.EntityRepository, serverID string, limit uint) (board []get.Entity, err error) {
	return repo.Leaderboard(serverID, limit)
}

// Loserboard queries the entities with the lowest karma levels (the number of
// entities returned will not exceed 'limit.')
func Loserboard(repo get.EntityRepository, serverID string, limit uint) (board []get.Entity, err error) {
	return repo.Loserboard(serverID, limit)
}
