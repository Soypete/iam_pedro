package database

import (
	"embed"
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

type Postgres struct {
	connections *sqlx.DB
	modelName   string
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

func NewPostgres() (Postgres, error) {
	dbx, err := sqlx.Connect("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		return Postgres{}, fmt.Errorf("error connecting to postgres: %w", err)
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return Postgres{}, fmt.Errorf("error setting dialect: %w", err)
	}

	if err := goose.Up(dbx.DB, "migrations"); err != nil {
		return Postgres{}, fmt.Errorf("error running migrations: %w", err)
	}

	if err := dbx.Ping(); err != nil {
		return Postgres{}, fmt.Errorf("error pinging postgres: %w", err)
	}

	return Postgres{
		connections: dbx,
		modelName:   "Mistral_7B_v0.1.4",
	}, nil
}

func (p Postgres) Close() {
	fmt.Println("Closing postgres connection")
	p.connections.Close()
}
