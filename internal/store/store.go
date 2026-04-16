package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Todo struct {
	ID          int64
	Title       string
	Description string
	Completed   bool
	Position    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS todos (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	completed INTEGER NOT NULL DEFAULT 0,
	position INTEGER NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create todos table: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) CreateTodo(ctx context.Context, title, description string) (Todo, error) {
	if title == "" {
		return Todo{}, errors.New("title is required")
	}

	var nextPosition int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), 0) + 1 FROM todos`).Scan(&nextPosition); err != nil {
		return Todo{}, fmt.Errorf("resolve todo position: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `
INSERT INTO todos (title, description, completed, position)
VALUES (?, ?, 0, ?)
`, title, description, nextPosition)
	if err != nil {
		return Todo{}, fmt.Errorf("insert todo: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Todo{}, fmt.Errorf("resolve inserted todo id: %w", err)
	}

	return s.GetTodo(ctx, id)
}

func (s *SQLiteStore) GetTodo(ctx context.Context, id int64) (Todo, error) {
	var todo Todo
	var completedInt int

	err := s.db.QueryRowContext(ctx, `
SELECT id, title, description, completed, position, created_at, updated_at
FROM todos
WHERE id = ?
`, id).Scan(
		&todo.ID,
		&todo.Title,
		&todo.Description,
		&completedInt,
		&todo.Position,
		&todo.CreatedAt,
		&todo.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Todo{}, fmt.Errorf("todo %d not found", id)
		}
		return Todo{}, fmt.Errorf("get todo: %w", err)
	}

	todo.Completed = completedInt == 1
	return todo, nil
}

func (s *SQLiteStore) ListTodos(ctx context.Context) ([]Todo, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, title, description, completed, position, created_at, updated_at
FROM todos
ORDER BY position ASC, id ASC
`)
	if err != nil {
		return nil, fmt.Errorf("list todos: %w", err)
	}
	defer rows.Close()

	todos := make([]Todo, 0)
	for rows.Next() {
		var todo Todo
		var completedInt int
		if err := rows.Scan(
			&todo.ID,
			&todo.Title,
			&todo.Description,
			&completedInt,
			&todo.Position,
			&todo.CreatedAt,
			&todo.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan todo: %w", err)
		}
		todo.Completed = completedInt == 1
		todos = append(todos, todo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate todos: %w", err)
	}

	return todos, nil
}
