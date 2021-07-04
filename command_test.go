package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const fixturesDir string = "test-fixtures"

func TestCheckKarma(t *testing.T) {
	interactiveCases := []struct {
		name    string
		input   request
		needles []string
	}{
		{"subject with pre-existing karma", request{message: "Popple"}, []string{hasKarma("Popple", 1)}},
		{"subject without karma", request{message: "Nobody"}, []string{hasKarma("Nobody", 0)}},
		{"multiple subjects", request{message: "Popple Nobody Gophers"}, []string{hasKarma("Nobody", 0), hasKarma("Popple", 1), hasKarma("Gophers", 12)}},
	}

	db, cleanup := makeScratchDB(t, []Entity{
		{Name: "Popple", Karma: 1},
		{Name: "Gophers", Karma: 12},
	})
	defer cleanup()

	for _, tt := range interactiveCases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			CheckKarma(tt.input, &rsp, db)
			assertNumResponses(t, rsp, 1)
			assertHasAllSubstrings(t, rsp.responses[0].value, tt.needles)
		})
	}

	ignoreCases := []struct {
		name  string
		input request
	}{
		{"in direct message context", request{message: "Popple", isDM: true}},
	}

	for _, tt := range ignoreCases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			CheckKarma(tt.input, &rsp, db)
			assertNumResponses(t, rsp, 0)
		})
	}
}

func assertNumResponses(t *testing.T, rsp responseSink, expected int) {
	if len(rsp.responses) != expected {
		t.Errorf("expected %d responses, got %d %#v", expected, len(rsp.responses), rsp)
	}
}

func assertHasAllSubstrings(t *testing.T, haystack string, needles []string) {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			t.Errorf("didn't find %q in %#v", needle, haystack)
		}
	}
}

type responseSink struct {
	responses []testResponse
}

func (r *responseSink) SendMessageToChannel(msg string) error {
	r.sink(responseChannelMessage, msg)
	return nil
}

func (r *responseSink) SendReply(msg string) error {
	r.sink(responseReply, msg)
	return nil
}

func (r *responseSink) React(emoji string) error {
	r.sink(responseEmoji, emoji)
	return nil
}

func (r *responseSink) sink(kind responseType, msg string) {
	r.responses = append(r.responses, testResponse{kind, msg})
}

type testResponse struct {
	kind  responseType
	value string
}

type responseType int

const (
	responseChannelMessage responseType = iota
	responseReply
	responseEmoji
)

func hasKarma(name string, karma int) string {
	return fmt.Sprintf("%s has %d karma", name, karma)
}

func makeScratchDB(t *testing.T, rows []Entity) (*gorm.DB, func()) {
	_ = os.MkdirAll(fixturesDir, 0755)

	f, err := ioutil.TempFile(fixturesDir, "db")
	if err != nil {
		t.Fatalf("%s", err)
	}

	dbName := f.Name()
	f.Close()

	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("%s", err)
	}

	db.AutoMigrate(&Entity{})

	for _, r := range rows {
		db.Create(&r)
	}

	return db, func() {
		os.Remove(dbName)
	}
}
