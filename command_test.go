package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
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

	db, cleanup := makeScratchDB(t)
	populateEntitiesInDB(db, []Entity{
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
		{"empty", request{message: ""}},
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

func TestModKarma(t *testing.T) {
	interactiveCases := []struct {
		name    string
		input   request
		needles []string
	}{
		{"basic increment", request{message: "Test++"}, []string{hasKarma("Test", 1)}},
		{"basic decrement", request{message: "Test--"}, []string{hasKarma("Test", -1)}},
		{"many operations", request{message: "NoKarma SomeKarma++ LessKarma-- NoMoreKarma"}, []string{hasKarma("SomeKarma", 1), hasKarma("LessKarma", -1)}},
	}

	for _, tt := range interactiveCases {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := makeScratchDB(t)
			defer cleanup()

			var rsp responseSink
			ModKarma(tt.input, &rsp, db)
			assertNumResponses(t, rsp, 1)
			assertHasAllSubstrings(t, rsp.responses[0].value, tt.needles)
		})
	}

	ignoreCases := []struct {
		name  string
		input request
	}{
		{"ignore in direct message context", request{isDM: true, message: "Test++"}},
		{"ignore net zero operations", request{message: "Test++ Test--"}},
	}

	for _, tt := range ignoreCases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			ModKarma(tt.input, &rsp, nil)
			assertNumResponses(t, rsp, 0)
		})
	}

	dataCases := []struct {
		name   string
		input  request
		before []Entity
		after  []Entity
	}{
		{"reducing to zero removes row", request{message: "Test--"}, []Entity{{Name: "Test", Karma: 1}}, []Entity{}},
		{"adjusting karma is saved to existing row", request{message: "Test++"}, []Entity{{Name: "Test", Karma: 1}}, []Entity{{Name: "Test", Karma: 2}}},
		{"the first increment adds a new row", request{message: "Test++"}, []Entity{}, []Entity{{Name: "Test", Karma: 1}}},
	}

	for _, tt := range dataCases {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := makeScratchDB(t)
			defer cleanup()
			populateEntitiesInDB(db, tt.before)

			var rsp responseSink
			ModKarma(tt.input, &rsp, db)

			var actual []Entity
			db.Find(&actual)

			assertDataChanged(t, actual, tt.after)
		})
	}
}

func TestSetAnnounce(t *testing.T) {
	interactiveCases := []struct {
		name              string
		input             request
		expectedResponses []testResponse
		before, after     Config
	}{
		{"on", request{message: "on"}, []testResponse{{kind: responseEmoji, value: "👍"}}, Config{}, Config{NoAnnounce: false}},
		{"off", request{message: "off"}, []testResponse{{kind: responseEmoji, value: "👍"}}, Config{}, Config{NoAnnounce: true}},
		{"yes", request{message: "yes"}, []testResponse{{kind: responseEmoji, value: "👍"}}, Config{}, Config{NoAnnounce: false}},
		{"no", request{message: "no"}, []testResponse{{kind: responseEmoji, value: "👍"}}, Config{}, Config{NoAnnounce: true}},
		{"invalid setting", request{message: "asdf"}, []testResponse{{kind: responseReply, value: "Announce settings are: \"yes\", \"no\", \"on\", \"off\""}}, Config{}, Config{}},
		{"empty", request{message: ""}, []testResponse{{kind: responseReply, value: "Announce settings are: \"yes\", \"no\", \"on\", \"off\""}}, Config{}, Config{}},
	}

	for _, tt := range interactiveCases {
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := makeScratchDB(t)
			defer cleanup()
			populateConfigsInDB(db, []Config{tt.before})

			var rsp responseSink
			SetAnnounce(tt.input, &rsp, db)

			reflect.DeepEqual(tt.expectedResponses, rsp.responses)

			var actual Config
			db.Where(&Config{}).First(&actual)
			if actual.NoAnnounce != tt.after.NoAnnounce {
				t.Errorf("expected NoAnnounce=%v got %v", tt.after.NoAnnounce, actual.NoAnnounce)
			}
		})
	}

	ignoreCases := []struct {
		name  string
		input request
	}{
		{"ignored in DM context", request{isDM: true, message: "on"}},
	}

	for _, tt := range ignoreCases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			SetAnnounce(tt.input, &rsp, nil)
			assertNumResponses(t, rsp, 0)
		})
	}
}

func TestSendHelp(t *testing.T) {
	cases := []struct {
		name              string
		input             request
		expectedResponses []testResponse
	}{
		{"help sends link to usage", request{}, []testResponse{{kind: responseChannelMessage, value: "Usage: https://github.com/connorkuehl/popple#usage"}}},
		{"help sends link to usage in DM context", request{isDM: true}, []testResponse{{kind: responseChannelMessage, value: "Usage: https://github.com/connorkuehl/popple#usage"}}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			SendHelp(tt.input, &rsp)
			if !reflect.DeepEqual(rsp.responses, tt.expectedResponses) {
				t.Errorf("expected %#v got %#v", tt.expectedResponses, rsp.responses)
			}
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

func assertDataChanged(t *testing.T, actual, expected []Entity) {
	if len(actual) != len(expected) {
		t.Errorf("number of actual results different from expected: actual = %#v expected = %#v", actual, expected)
	}

	var expectMap = make(map[string]Entity)
	for _, a := range actual {
		expectMap[a.Name] = a
	}

	for _, e := range expected {
		row, ok := expectMap[e.Name]
		if !ok {
			t.Errorf("didn't find expected row %#v in %#v", e, expected)
		}
		if row.Karma != e.Karma {
			t.Errorf("wrong karma value for %#v - got %d, want %d", e, row.Karma, e.Karma)
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

func makeScratchDB(t *testing.T) (*gorm.DB, func()) {
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

	db.AutoMigrate(&Entity{}, &Config{})

	return db, func() {
		os.Remove(dbName)
	}
}

func populateEntitiesInDB(db *gorm.DB, rows []Entity) {
	for _, r := range rows {
		db.Create(&r)
	}
}

func populateConfigsInDB(db *gorm.DB, rows []Config) {
	for _, c := range rows {
		db.Create(&c)
	}
}
