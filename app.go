package popple

import (
	"time"

	"gorm.io/gorm"
)

// App is an instance of a Popple bot.
type App struct {
	db     *gorm.DB
	Router *Router
}

// NewApp creates a new Popple instance.
func NewApp(db *gorm.DB, start time.Time) (*App, error) {
	if err := db.AutoMigrate(&Config{}); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Entity{}); err != nil {
		return nil, err
	}

	var r Router
	r.addRoute("announce", func(req Request, rsp ResponseWriter) {
		SetAnnounce(req, rsp, db)
	})
	r.addRoute("help", func(req Request, rsp ResponseWriter) {
		SendHelp(req, rsp)
	})
	r.addRoute("karma", func(req Request, rsp ResponseWriter) {
		CheckKarma(req, rsp, db)
	})
	r.addRoute("bot", func(req Request, rsp ResponseWriter) {
		Bot(req, rsp, db)
	})
	r.addRoute("top", func(req Request, rsp ResponseWriter) {
		Top(req, rsp, db)
	})
	r.addRoute("uptime", func(req Request, rsp ResponseWriter) {
		Uptime(req, rsp, start)
	})
	r.addRoute("version", func(req Request, rsp ResponseWriter) {
		SendVersion(req, rsp)
	})

	// just check for karma operations by default if no other commands
	// were matched
	r.addRoute("*", func(req Request, rsp ResponseWriter) {
		ModKarma(req, rsp, db)
	})

	return &App{
		db:     db,
		Router: &r,
	}, nil
}
