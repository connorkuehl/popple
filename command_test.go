package popple

import (
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"github.com/connorkuehl/popple/internal/data"
	"github.com/connorkuehl/popple/internal/sql"
	"github.com/connorkuehl/popple/mocks"
)

var (
	entityColumns = []string{"id", "created_at", "updated_at", "karma"}
	configColumns = []string{"id", "created_at", "updated_at", "no_announce"}
)

func TestModKarma(t *testing.T) {
	noOps := []struct {
		name  string
		input Request
	}{
		{"direct message is a no-op", Request{IsDM: true, GuildID: "1", Message: "asdf++"}},
		{"net-zero karma is a no-op", Request{GuildID: "2", Message: "a++ a--"}},
	}

	for _, tt := range noOps {
		t.Run(tt.name, func(t *testing.T) {
			db, _, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			var rsp mocks.ResponseWriter

			// The sqlmock or ResponseWriter mock will panic and thus
			// fail the test if the code actually tries to act on this
			// when it shouldn't.
			ModKarma(tt.input, &rsp, db)
		})
	}

	t.Run("no message is sent when announcements are disabled", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		assert.NoError(t, err)
		defer db.Close()

		var rsp mocks.ResponseWriter
		req := Request{GuildID: "3", Message: "hello++"}

		mock.ExpectQuery(sql.GetEntity).
			WithArgs("hello", "3").
			WillReturnRows(mock.NewRows(entityColumns))

		mock.ExpectQuery(sql.GetConfig).
			WithArgs("3").
			// I'm not in love with using time.Now() but none of the
			// logic actually cares about these values. Those are just
			// there for the database administrator.
			WillReturnRows(mock.NewRows(configColumns).AddRow(1, time.Now(), time.Now(), true))

		// The ResponseWriter mock will panic if a message is sent.
		ModKarma(req, &rsp, db)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("entity is deleted when karma reaches zero", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		assert.NoError(t, err)
		defer db.Close()

		var rsp mocks.ResponseWriter
		req := Request{GuildID: "4", Message: "c++--"}

		mock.ExpectQuery(sql.GetEntity).
			WithArgs("c++", "4").
			WillReturnRows(mock.NewRows(entityColumns).AddRow(1, time.Now(), time.Now(), 1))

		mock.ExpectBegin()
		mock.ExpectExec(sql.DeleteEntity).
			WithArgs("c++", "4").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		mock.ExpectQuery(sql.GetConfig).
			WithArgs("4").
			WillReturnRows(mock.NewRows(configColumns).AddRow(1, time.Now(), time.Now(), true))

		ModKarma(req, &rsp, db)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("entity with non-zero karma is updated in db", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		assert.NoError(t, err)
		defer db.Close()

		var rsp mocks.ResponseWriter
		req := Request{GuildID: "5", Message: "c++"}

		mock.ExpectQuery(sql.GetEntity).
			WithArgs("c", "5").
			WillReturnRows(mock.NewRows(entityColumns).AddRow(1, time.Now(), time.Now(), 8))

		mock.ExpectBegin()
		mock.ExpectExec(sql.CreateEntity).
			WithArgs("c", "5").
			WillReturnError(errors.New("already exists"))
		mock.ExpectExec(sql.PutEntity).
			WithArgs(9, "c", "5").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		mock.ExpectQuery(sql.GetConfig).
			WithArgs("5").
			WillReturnRows(mock.NewRows(configColumns).AddRow(1, time.Now(), time.Now(), true))

		ModKarma(req, &rsp, db)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCheckKarma(t *testing.T) {
	t.Run("direct message is a no-op", func(t *testing.T) {
		db, _, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		var rsp mocks.ResponseWriter
		req := Request{IsDM: true, Message: "personA personB"}

		CheckKarma(req, &rsp, db)
	})

	t.Run("no subjects is a no-op", func(t *testing.T) {
		db, _, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		req := Request{Message: ""}
		var rsp mocks.ResponseWriter

		CheckKarma(req, &rsp, db)
	})

	t.Run("subjects not in the db have zero karma", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		assert.NoError(t, err)
		defer db.Close()

		req := Request{GuildID: "6", Message: "ghost"}
		var rsp mocks.ResponseWriter

		rsp.On("SendMessageToChannel", "ghost has 0 karma.").Return(nil)

		mock.ExpectQuery(sql.GetEntity).
			WithArgs("ghost", "6").
			WillReturnRows(mock.NewRows(entityColumns))

		CheckKarma(req, &rsp, db)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSetAnnounce(t *testing.T) {
	t.Run("direct message is a no-op", func(t *testing.T) {
		db, _, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		var rsp mocks.ResponseWriter
		req := Request{IsDM: true, Message: "on"}

		SetAnnounce(req, &rsp, db)
	})

	successCases := []struct {
		verb string
		want bool
	}{
		{"on", false},
		{"yes", false},
		{"off", true},
		{"no", true},
	}

	for _, tt := range successCases {
		t.Run(tt.verb, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer db.Close()

			req := Request{GuildID: "7", Message: tt.verb}

			var rsp mocks.ResponseWriter
			rsp.On("React", "👍").Return(nil)

			mock.ExpectQuery(sql.GetConfig).
				WithArgs("7").
				WillReturnRows(mock.NewRows(configColumns))

			mock.ExpectBegin()
			mock.ExpectExec(sql.CreateConfig).
				WithArgs("7").
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(sql.PutConfig).
				WithArgs(tt.want, "7").
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			SetAnnounce(req, &rsp, db)

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("replies with an error message if invalid option is used", func(t *testing.T) {
		db, _, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		req := Request{GuildID: "8", Message: "hjkl"}

		var rsp mocks.ResponseWriter
		rsp.On("SendReply", "Announce settings are: \"yes\", \"no\", \"on\", \"off\"").
			Return(nil)

		SetAnnounce(req, &rsp, db)
	})
}

func TestBoard(t *testing.T) {
	t.Run("limit less than 1 is a no-op", func(t *testing.T) {
		db, _, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		req := Request{GuildID: "9", Message: "-1"}

		var rsp mocks.ResponseWriter

		board(req, &rsp, db, data.Ascending)
	})

	t.Run("nothing to display is a no-op", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		assert.NoError(t, err)
		defer db.Close()

		req := Request{GuildID: "10", Message: "5"}

		var rsp mocks.ResponseWriter

		columns := []string{"id", "created_at", "updated_at", "name", "server_id", "karma"}

		// GetTopEntities is the default behavior (see Top and Bot).
		mock.ExpectQuery(sql.GetTopEntities).
			WithArgs("10", 5).
			WillReturnRows(mock.NewRows(columns))

		// data.Ascending because we are using sql.GetTopEntities
		board(req, &rsp, db, data.Ascending)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
