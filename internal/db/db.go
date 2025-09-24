package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return sqlDB, nil
}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  first_name TEXT    NOT NULL,
  last_name  TEXT    NOT NULL,
  email      TEXT    NOT NULL UNIQUE,
  password   TEXT    NOT NULL,
  usm_pesos  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS books (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  book_name        TEXT    NOT NULL,
  book_category    TEXT    NOT NULL,
  transaction_type TEXT    NOT NULL CHECK (transaction_type IN ('Venta','Arriendo')),
  price            INTEGER NOT NULL,
  popularity_score INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS inventory (
  book_id            INTEGER PRIMARY KEY,
  available_quantity INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(book_id) REFERENCES books(id) ON DELETE CASCADE
);

-- ventas (compra de 1 libro por registro)
CREATE TABLE IF NOT EXISTS sales (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id   INTEGER NOT NULL,
  book_id   INTEGER NOT NULL,
  sale_date TEXT    NOT NULL, -- DD/MM/YYYY
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(book_id) REFERENCES books(id)
);

-- prestamos (lo implementamos después)
CREATE TABLE IF NOT EXISTS loans (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id     INTEGER NOT NULL,
  book_id     INTEGER NOT NULL,
  start_date  TEXT    NOT NULL, -- DD/MM/YYYY
  return_date TEXT,              -- DD/MM/YYYY o NULL
  status      TEXT    NOT NULL CHECK (status IN ('pendiente','finalizado')),
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(book_id) REFERENCES books(id)
);
`)
	return err
}
