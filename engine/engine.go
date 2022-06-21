package engine

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Engine struct {
	readDB   *sql.DB
	writeDB  *sql.DB
	readOnly bool
	isMem    bool
	path     string
}

// Open creates a new database engine instance. It starts a read-only and a write connection.
func Open(path string) (*Engine, error) {
	// write and read database connection
	writeDB, err := sql.Open("sqlite3", fmt.Sprintf("file:%s", path))
	if err != nil {
		return nil, err
	}

	// read-only database connection
	readDB, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", path))
	if err != nil {
		return nil, err
	}

	if err := writeDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed database ping: %s", err)
	}

	return &Engine{
		readDB:   readDB,
		writeDB:  writeDB,
		path:     path,
		isMem:    false,
		readOnly: false,
	}, nil
}

func (eng *Engine) Path() string {
	return eng.path
}

func (eng *Engine) IsMem() bool {
	return eng.isMem
}

func (eng *Engine) IsReadOnly() bool {
	return eng.readOnly
}

// Close the database connections
func (eng *Engine) Close() error {
	if err := eng.readDB.Close(); err != nil {
		return err
	}

	return eng.writeDB.Close()
}
