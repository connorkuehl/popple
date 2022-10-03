package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-sql-driver/mysql"
	"github.com/sethvargo/go-retry"
	"github.com/sirupsen/logrus"

	"github.com/connorkuehl/popple/cmd/popplesvc/internal/popple"
	"github.com/connorkuehl/popple/cmd/popplesvc/internal/popplehttp"
	"github.com/connorkuehl/popple/cmd/popplesvc/internal/store"
)

var (
	dbHost = os.Getenv("POPPLE_DB_HOST")
	dbPort = os.Getenv("POPPLE_DB_PORT")
	dbUser = os.Getenv("POPPLE_DB_USER")
	dbPass = os.Getenv("POPPLE_DB_PASS")
	dbName = os.Getenv("POPPLE_DB_NAME")
)

var log = logrus.StandardLogger().WithFields(logrus.Fields{
	"component": "svc",
})

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dbcfg := mysql.Config{
		User:                 dbUser,
		Passwd:               dbPass,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", dbHost, dbPort),
		DBName:               dbName,
		ParseTime:            true,
		AllowNativePasswords: true,
	}

	connCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var db *sql.DB
	err := retry.Fibonacci(connCtx, 1*time.Second, func(ctx context.Context) error {
		log.Info("connecting to MySQL database")

		var err error
		db, err = sql.Open("mysql", dbcfg.FormatDSN())
		if err != nil {
			log.WithError(err).Error("failed to connect to db,", err, "retrying")
			return retry.RetryableError(err)
		}

		err = db.PingContext(ctx)
		if err != nil {
			log.WithError(err).Error("database ping failed")
		}
		return retry.RetryableError(err)
	})
	if err != nil {
		return err
	}
	defer db.Close()

	mysqlStore := store.NewMySQLStore(db)
	svc := popple.NewService(mysqlStore)
	server := popplehttp.NewServer(svc)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/boards/{server_id}", server.Board)
	router.Get("/configs/{server_id}", server.Config)
	router.Post("/configs/{server_id}", server.PutConfig)
	router.Post("/counts/{server_id}", server.ChangeKarma)
	router.Get("/counts/{server_id}", server.CheckKarma)

	log.Info("ready to serve requests")
	return http.ListenAndServe(":8080", router)
}
