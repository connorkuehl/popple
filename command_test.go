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
		{"subject with pre-existing karma", request{message: "Popple"}, []string{testFormatKarmaStatement("Popple", 1)}},
		{"subject without karma", request{message: "Nobody"}, []string{testFormatKarmaStatement("Nobody", 0)}},
		{"multiple subjects", request{message: "Popple Nobody Gophers"}, []string{testFormatKarmaStatement("Nobody", 0), testFormatKarmaStatement("Popple", 1), testFormatKarmaStatement("Gophers", 12)}},
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
		{"basic increment", request{message: "Test++"}, []string{testFormatKarmaStatement("Test", 1)}},
		{"basic decrement", request{message: "Test--"}, []string{testFormatKarmaStatement("Test", -1)}},
		{"many operations", request{message: "NoKarma SomeKarma++ LessKarma-- NoMoreKarma"}, []string{testFormatKarmaStatement("SomeKarma", 1), testFormatKarmaStatement("LessKarma", -1)}},
		{"a paren subject can have a leading @", request{message: "(@holdo)++"}, []string{testFormatKarmaStatement("@holdo", 1)}},
		{"a plaintext subject has @ prefix stripped", request{message: "@holdo++"}, []string{testFormatKarmaStatement("holdo", 1)}},
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

func TestSendVersion(t *testing.T) {
	cases := []struct {
		name              string
		input             request
		expectedResponses []testResponse
	}{
		{"version sends version", request{}, []testResponse{{kind: responseChannelMessage, value: fmt.Sprintf("I'm running version %s.", Version)}}},
		{"version sends version in DM context", request{isDM: true}, []testResponse{{kind: responseChannelMessage, value: fmt.Sprintf("I'm running version %s.", Version)}}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			SendVersion(tt.input, &rsp)

			if !reflect.DeepEqual(rsp.responses, tt.expectedResponses) {
				t.Errorf("expected %#v got %#v", tt.expectedResponses, rsp.responses)
			}
		})
	}
}

func TestTop(t *testing.T) {
	db, cleanup := makeScratchDB(t)
	defer cleanup()

	populateEntitiesInDB(db, []Entity{
		{Name: "A", Karma: 10},
		{Name: "B", Karma: 9},
		{Name: "C", Karma: 8},
		{Name: "D", Karma: 7},
		{Name: "E", Karma: 6},
		{Name: "F", Karma: 5},
		{Name: "G", Karma: 4},
		{Name: "H", Karma: 3},
		{Name: "I", Karma: 2},
		{Name: "J", Karma: 1},
		{Name: "K", Karma: 0},
	})

	cases := []struct {
		name     string
		input    request
		expected testResponse
	}{
		{"returns the top 3", request{message: "3"}, testResponse{responseChannelMessage, entityToLeaderboard([]Entity{
			{Name: "A", Karma: 10},
			{Name: "B", Karma: 9},
			{Name: "C", Karma: 8},
		})}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			Top(tt.input, &rsp, db)

			assertNumResponses(t, rsp, 1)
			if tt.expected != rsp.responses[0] {
				t.Errorf("got %s want %s", rsp.responses[0].value, tt.expected.value)
			}
		})
	}

	ignoreCases := []struct {
		name  string
		input request
	}{
		{"in a DM context", request{isDM: true}},
		{"zero limit", request{message: "0"}},
		{"negative limit", request{message: "-1"}},
	}

	for _, tt := range ignoreCases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			Top(tt.input, &rsp, db)
			assertNumResponses(t, rsp, 0)
		})
	}
}

func TestBot(t *testing.T) {
	db, cleanup := makeScratchDB(t)
	defer cleanup()

	populateEntitiesInDB(db, []Entity{
		{Name: "A", Karma: 10},
		{Name: "B", Karma: 9},
		{Name: "C", Karma: 8},
		{Name: "D", Karma: 7},
		{Name: "E", Karma: 6},
		{Name: "F", Karma: 5},
		{Name: "G", Karma: 4},
		{Name: "H", Karma: 3},
		{Name: "I", Karma: 2},
		{Name: "J", Karma: 1},
		{Name: "K", Karma: 0},
	})

	cases := []struct {
		name     string
		input    request
		expected testResponse
	}{
		{"returns the bottom 3", request{message: "3"}, testResponse{responseChannelMessage, entityToLeaderboard([]Entity{
			{Name: "K", Karma: 0},
			{Name: "J", Karma: 1},
			{Name: "I", Karma: 2},
		})}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			Bot(tt.input, &rsp, db)

			assertNumResponses(t, rsp, 1)
			if tt.expected != rsp.responses[0] {
				t.Errorf("got %s want %s", rsp.responses[0].value, tt.expected.value)
			}
		})
	}

	ignoreCases := []struct {
		name  string
		input request
	}{
		{"in a DM context", request{isDM: true}},
		{"zero limit", request{message: "0"}},
		{"negative limit", request{message: "-1"}},
	}

	for _, tt := range ignoreCases {
		t.Run(tt.name, func(t *testing.T) {
			var rsp responseSink
			Top(tt.input, &rsp, db)
			assertNumResponses(t, rsp, 0)
		})
	}

}

func TestRouter(t *testing.T) {
	const bot string = "@Popple"

	cases := []struct {
		name   string
		req    request
		routes []route
	}{
		{"no routes", request{message: "asdf"}, []route{}},
		{"catchall", request{}, []route{
			{"help", func(req request, rsp responseWriter) {
				t.Errorf("expected to be routed to catchall, but wasn't")
			}},
			{"*", func(req request, rsp responseWriter) {
				// yay
			}},
		}},
		{"username and command is stripped", request{message: bot + " help pass"}, []route{
			{"help", func(req request, rsp responseWriter) {
				if req.message != "pass" {
					t.Errorf("got %s, want %s", req.message, "pass")
				}
			}},
			{"*", func(req request, rsp responseWriter) {
				t.Errorf("fell into catchall, should have been routed elsewhere")
			}},
		}},
		{"username is required outside of DMs", request{message: "help"}, []route{
			{"help", func(req request, rsp responseWriter) {
				t.Errorf("made it to subcommand but bot wasn't mentioned")
			}},
			{"*", func(req request, rsp responseWriter) {
				// yay
			}},
		}},
		{"username is optional in DMs", request{message: "help", isDM: true}, []route{
			{"help", func(req request, rsp responseWriter) {
				// yay
			}},
			{"*", func(req request, rsp responseWriter) {
				t.Errorf("fell into catchall, should have been routed elsewhere")
			}},
		}},
		{"can use username in DMs if preferred", request{message: bot + " help", isDM: true}, []route{
			{"help", func(req request, rsp responseWriter) {
				// yay
			}},
			{"*", func(req request, rsp responseWriter) {
				t.Errorf("fell into catchall, should have been routed elsewhere")
			}},
		}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := router{}
			r.bot = bot
			for _, route := range tt.routes {
				r.addRoute(route.match, route.cmd)
			}

			r.route(tt.req, nil)
		})
	}
}

func entityToLeaderboard(entities []Entity) string {
	builder := strings.Builder{}
	for _, e := range entities {
		builder.WriteString(testFormatKarmaLeaderboardEntry(e.Name, e.Karma))
	}
	return builder.String()
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

	_ = db.AutoMigrate(&Entity{}, &Config{})

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
