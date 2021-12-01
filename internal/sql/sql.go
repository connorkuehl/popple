package sql

import (
	_ "embed"
)

var (
	//go:embed make_tables.sql
	MakeTables string

	//go:embed create_config.sql
	CreateConfig string

	//go:embed get_config.sql
	GetConfig string

	//go:embed put_config.sql
	PutConfig string

	//go:embed create_entity.sql
	CreateEntity string

	//go:embed delete_entity.sql
	DeleteEntity string

	//go:embed get_entity.sql
	GetEntity string

	//go:embed get_top_entities.sql
	GetTopEntities string

	//go:embed get_bot_entities.sql
	GetBotEntities string

	//go:embed put_entity.sql
	PutEntity string
)
