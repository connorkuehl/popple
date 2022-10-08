package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/connorkuehl/popple"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{
		db: db,
	}
}

func (s *MySQLStore) Board(ctx context.Context, serverID string, ord popple.BoardOrder, limit uint) (popple.Board, error) {
	const (
		mysqlBoardAsc  = `SELECT name, karma FROM entities WHERE server_id=? ORDER BY karma ASC LIMIT ?`
		mysqlBoardDesc = `SELECT name, karma FROM entities WHERE server_id=? ORDER BY karma DESC LIMIT ?`
	)

	ordSQL := mysqlBoardAsc
	switch ord {
	case popple.BoardOrderAsc:
	case popple.BoardOrderDsc:
		ordSQL = mysqlBoardDesc
	default:
		return nil, errors.New("invalid order")
	}

	rows, err := s.db.QueryContext(ctx, ordSQL, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var board popple.Board
	for rows.Next() {
		var entry popple.BoardEntry
		err := rows.Scan(&entry.Who, &entry.Karma)
		if err != nil {
			return nil, err
		}

		board = append(board, entry)
	}

	return board, nil
}

func (s *MySQLStore) ChangeKarma(ctx context.Context, serverID string, increments popple.Increments) (popple.Increments, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const changeKarmaSQL = `
	INSERT INTO entities (
		server_id,
		name,
		karma
	) VALUES (
		?,
		?,
		karma+?
	) ON DUPLICATE KEY UPDATE updated_at=NOW(), karma=karma+?`

	for who, karma := range increments {
		_, err := tx.ExecContext(ctx, changeKarmaSQL, serverID, who, karma, karma)
		if err != nil {
			return nil, err
		}
	}

	const mysqlGetKarma = `SELECT karma FROM entities WHERE server_id=? AND name=?`

	newLevels := make(popple.Increments)
	var garbageCollect []string
	for who := range increments {
		var karma int64
		err := tx.QueryRowContext(ctx, mysqlGetKarma, serverID, who).Scan(&karma)
		if err != nil {
			return nil, err
		}

		newLevels[who] = karma
		if karma == 0 {
			garbageCollect = append(garbageCollect, who)
		}
	}

	const mysqlRemoveSubject = `DELETE FROM entities WHERE server_id=? AND name=?`

	for _, who := range garbageCollect {
		_, err := tx.ExecContext(ctx, mysqlRemoveSubject, serverID, who)
		if err != nil {
			return nil, err
		}
	}

	return newLevels, tx.Commit()
}

func (s *MySQLStore) CheckKarma(ctx context.Context, serverID string, who []string) (map[string]int64, error) {
	increments := make(map[string]int64)

	const mysqlCheckKarma = `SELECT karma FROM entities WHERE server_id=? AND name=?`

	for _, name := range who {
		var karma int64
		err := s.db.QueryRowContext(ctx, mysqlCheckKarma, serverID, name).Scan(&karma)
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
		}
		if err != nil {
			return nil, err
		}

		increments[name] = karma
	}

	return increments, nil
}

func (s *MySQLStore) Config(ctx context.Context, serverID string) (*popple.Config, error) {
	cfg := popple.Config{ServerID: serverID}

	const mysqlGetConfig = `SELECT no_announce FROM configs WHERE server_id=?`

	err := s.db.QueryRowContext(ctx, mysqlGetConfig, serverID).Scan(&cfg.NoAnnounce)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (s *MySQLStore) PutConfig(ctx context.Context, config *popple.Config) error {
	const mysqlPutConfig = `
	INSERT INTO configs (
		server_id,
		no_announce
	) VALUES (
		?,
		?
	) ON DUPLICATE KEY UPDATE updated_at=NOW(), no_announce=?`

	_, err := s.db.ExecContext(ctx, mysqlPutConfig, config.ServerID, config.NoAnnounce, config.NoAnnounce)
	if err != nil {
		return err
	}

	return nil
}
