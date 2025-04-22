package database

import (
	"embed"
	"fmt"
	"os"

	"github.com/Soypete/twitch-llm-bot/logging"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

type Postgres struct {
	connections *sqlx.DB
	logger      *logging.Logger
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

func NewPostgres(logger *logging.Logger) (*Postgres, error) {
	if logger == nil {
		logger = logging.Default()
	}

	logger.Info("connecting to postgres database")
	dbURL := os.Getenv("POSTGRES_URL")

	dbx, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		logger.Error("error connecting to postgres", "error", err.Error())
		return nil, fmt.Errorf("error connecting to postgres: %w", err)
	}

	logger.Debug("setting up migration system")
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Error("error setting dialect", "error", err.Error())
		return nil, fmt.Errorf("error setting dialect: %w", err)
	}

	// TODO: do not commit
	if err := goose.DownTo(dbx.DB, "migrations", 4); err != nil  {
		return nil, fmt.Errorf("error running down migrations: %w", err)
	}

	logger.Info("running database migrations")
	if err := goose.Up(dbx.DB, "migrations"); err != nil {
		logger.Error("error running migrations", "error", err.Error())
		return nil, fmt.Errorf("error running migrations: %w", err)
	}

	logger.Debug("verifying database connection")
	if err := dbx.Ping(); err != nil {
		logger.Error("error pinging postgres", "error", err.Error())
		return nil, fmt.Errorf("error pinging postgres: %w", err)
	}

	logger.Info("database connection established successfully")
	return &Postgres{
		connections: dbx,
		logger:      logger,
	}, nil
}

func (p Postgres) Close() {
	p.logger.Info("closing postgres connection")
	p.connections.Close()
}
