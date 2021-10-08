package popple

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
		input   Request
		needles []string
	}{
		{"subject with pre-existing karma", Request{Message: "Popple"}, []string{testFormatKarmaStatement("Popple", 1)}},
		{"subject without karma", Request{Message: "Nobody"}, []string{testFormatKarmaStatement("Nobody", 0)}},
		{"multiple subjects", Request{Message: "Popple Nobody Gophers"}, []string{testFormatKarmaStatement("Nobody", 0), testFormatKarmaStatement("Popple", 1), testFormatKarmaStatement("Gophers", 12)}},
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
		input Request
	}{
		{"empty", Request{Message: ""}},
		{"in direct message context", Request{Message: "Popple", IsDM: true}},
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
		input   Request
		needles []string
	}{
		{"basic increment", Request{Message: "Test++"}, []string{testFormatKarmaStatement("Test", 1)}},
		{"basic decrement", Request{Message: "Test--"}, []string{testFormatKarmaStatement("Test", -1)}},
		{"many operations", Request{Message: "NoKarma SomeKarma++ LessKarma-- NoMoreKarma"}, []string{testFormatKarmaStatement("SomeKarma", 1), testFormatKarmaStatement("LessKarma", -1)}},
		{"a paren subject can have a leading @", Request{Message: "(@holdo)++"}, []string{testFormatKarmaStatement("@holdo", 1)}},
		{"a plaintext subject has @ prefix stripped", Request{Message: "@holdo++"}, []string{testFormatKarmaStatement("holdo", 1)}},
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
		input Request
	}{
		{"ignore in direct message context", Request{IsDM: true, Message: "Test++"}},
		{"ignore net zero operations", Request{Message: "Test++ Test--"}},
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
		input  Request
		before []Entity
		after  []Entity
	}{
		{"reducing to zero removes row", Request{Message: "Test--"}, []Entity{{Name: "Test", Karma: 1}}, []Entity{}},
		{"adjusting karma is saved to existing row", Request{Message: "Test++"}, []Entity{{Name: "Test", Karma: 1}}, []Entity{{Name: "Test", Karma: 2}}},
		{"the first increment adds a new row", Request{Message: "Test++"}, []Entity{}, []Entity{{Name: "Test", Karma: 1}}},
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
		input             Request
		expectedResponses []testResponse
		before, after     Config
	}{
		{"on", Request{Message: "on"}, []testResponse{{kind: responseEmoji, value: "👍"}}, Config{}, Config{NoAnnounce: false}},
		{"off", Request{Message: "off"}, []testResponse{{kind: responseEmoji, value: "👍"}}, Config{}, Config{NoAnnounce: true}},
		{"yes", Request{Message: "yes"}, []testResponse{{kind: responseEmoji, value: "👍"}}, Config{}, Config{NoAnnounce: false}},
		{"no", Request{Message: "no"}, []testResponse{{kind: responseEmoji, value: "👍"}}, Config{}, Config{NoAnnounce: true}},
		{"invalid setting", Request{Message: "asdf"}, []testResponse{{kind: responseReply, value: "Announce settings are: \"yes\", \"no\", \"on\", \"off\""}}, Config{}, Config{}},
		{"empty", Request{Message: ""}, []testResponse{{kind: responseReply, value: "Announce settings are: \"yes\", \"no\", \"on\", \"off\""}}, Config{}, Config{}},
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
		input Request
	}{
		{"ignored in DM context", Request{IsDM: true, Message: "on"}},
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
		input             Request
		expectedResponses []testResponse
	}{
		{"help sends link to usage", Request{}, []testResponse{{kind: responseChannelMessage, value: "Usage: https://github.com/connorkuehl/popple#usage"}}},
		{"help sends link to usage in DM context", Request{IsDM: true}, []testResponse{{kind: responseChannelMessage, value: "Usage: https://github.com/connorkuehl/popple#usage"}}},
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
		input             Request
		expectedResponses []testResponse
	}{
		{"version sends version", Request{}, []testResponse{{kind: responseChannelMessage, value: fmt.Sprintf("I'm running version %s.", Version)}}},
		{"version sends version in DM context", Request{IsDM: true}, []testResponse{{kind: responseChannelMessage, value: fmt.Sprintf("I'm running version %s.", Version)}}},
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
		{Name: "K", Karma: 0},
		{Name: "J", Karma: 1},
	})

	cases := []struct {
		name     string
		input    Request
		expected testResponse
	}{
		{"returns the top 3", Request{Message: "3"}, testResponse{responseChannelMessage, entityToLeaderboard([]Entity{
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
		input Request
	}{
		{"in a DM context", Request{IsDM: true}},
		{"zero limit", Request{Message: "0"}},
		{"negative limit", Request{Message: "-1"}},
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
		input    Request
		expected testResponse
	}{
		{"returns the bottom 3", Request{Message: "3"}, testResponse{responseChannelMessage, entityToLeaderboard([]Entity{
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
		input Request
	}{
		{"in a DM context", Request{IsDM: true}},
		{"zero limit", Request{Message: "0"}},
		{"negative limit", Request{Message: "-1"}},
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
		req    Request
		routes []route
	}{
		{"no routes", Request{Message: "asdf"}, []route{}},
		{"catchall", Request{}, []route{
			{"help", func(req Request, rsp ResponseWriter) {
				t.Errorf("expected to be routed to catchall, but wasn't")
			}},
			{"*", func(req Request, rsp ResponseWriter) {
				// yay
			}},
		}},
		{"username and command is stripped", Request{Message: bot + " help pass"}, []route{
			{"help", func(req Request, rsp ResponseWriter) {
				if req.Message != "pass" {
					t.Errorf("got %s, want %s", req.Message, "pass")
				}
			}},
			{"*", func(req Request, rsp ResponseWriter) {
				t.Errorf("fell into catchall, should have been routed elsewhere")
			}},
		}},
		{"username is required outside of DMs", Request{Message: "help"}, []route{
			{"help", func(req Request, rsp ResponseWriter) {
				t.Errorf("made it to subcommand but bot wasn't mentioned")
			}},
			{"*", func(req Request, rsp ResponseWriter) {
				// yay
			}},
		}},
		{"username is optional in DMs", Request{Message: "help", IsDM: true}, []route{
			{"help", func(req Request, rsp ResponseWriter) {
				// yay
			}},
			{"*", func(req Request, rsp ResponseWriter) {
				t.Errorf("fell into catchall, should have been routed elsewhere")
			}},
		}},
		{"can use username in DMs if preferred", Request{Message: bot + " help", IsDM: true}, []route{
			{"help", func(req Request, rsp ResponseWriter) {
				// yay
			}},
			{"*", func(req Request, rsp ResponseWriter) {
				t.Errorf("fell into catchall, should have been routed elsewhere")
			}},
		}},
		{"commands must be individual word", Request{Message: bot + " helpasdf"}, []route{
			{"help", func(req Request, rsp ResponseWriter) {
				t.Errorf("should have fallen into catchall helpasdf != help: %#v", req)
			}},
			{"*", func(req Request, rsp ResponseWriter) {
				// yay
			}},
		}},
		{"commands must be individual word in DMs", Request{Message: " helpasdf", IsDM: true}, []route{
			{"help", func(req Request, rsp ResponseWriter) {
				t.Errorf("should have fallen into catchall helpasdf != help: %#v", req)
			}},
			{"*", func(req Request, rsp ResponseWriter) {
				// yay
			}},
		}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := Router{}
			r.Bot = bot
			for _, route := range tt.routes {
				r.addRoute(route.match, route.cmd)
			}

			r.Route(tt.req, nil)
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
