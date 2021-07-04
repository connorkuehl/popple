package main

import (
	"io/ioutil"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const fixturesDir string = "test-fixtures"

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
