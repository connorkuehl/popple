module github.com/connorkuehl/popple

go 1.17

require (
	github.com/bwmarrin/discordgo v0.23.2
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/jinzhu/gorm v1.9.16
	github.com/mattn/go-sqlite3 v1.14.8 // indirect
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.21.12
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.2 // indirect
)

replace github.com/bwmarrin/discordgo => github.com/connorkuehl/discordgo v0.23.3-0.20210822184312-e9475c4c43f4
