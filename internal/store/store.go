package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Todo struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
	title = strings.TrimSpace(title)
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
	row := s.db.QueryRowContext(ctx, `
SELECT id, title, description, completed, position, created_at, updated_at
FROM todos
WHERE id = ?
`, id)

	return scanTodoRow(row)
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

func (s *SQLiteStore) UpdateTodo(ctx context.Context, id int64, title, description string) (Todo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Todo{}, errors.New("title is required")
	}

	result, err := s.db.ExecContext(ctx, `
UPDATE todos
SET title = ?, description = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, title, description, id)
	if err != nil {
		return Todo{}, fmt.Errorf("update todo: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Todo{}, fmt.Errorf("resolve updated todo rows: %w", err)
	}
	if affected == 0 {
		return Todo{}, fmt.Errorf("todo %d not found", id)
	}

	return s.GetTodo(ctx, id)
}

func (s *SQLiteStore) SetCompleted(ctx context.Context, id int64, completed bool) (Todo, error) {
	completedInt := 0
	if completed {
		completedInt = 1
	}

	result, err := s.db.ExecContext(ctx, `
UPDATE todos
SET completed = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, completedInt, id)
	if err != nil {
		return Todo{}, fmt.Errorf("set todo completion: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Todo{}, fmt.Errorf("resolve updated todo rows: %w", err)
	}
	if affected == 0 {
		return Todo{}, fmt.Errorf("todo %d not found", id)
	}

	return s.GetTodo(ctx, id)
}

func (s *SQLiteStore) DeleteTodo(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start delete transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `DELETE FROM todos WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete todo: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("resolve deleted todo rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("todo %d not found", id)
	}

	if _, err := tx.ExecContext(ctx, `
WITH ordered AS (
	SELECT id, ROW_NUMBER() OVER (ORDER BY position ASC, id ASC) AS next_position
	FROM todos
)
UPDATE todos
SET position = (SELECT next_position FROM ordered WHERE ordered.id = todos.id),
	updated_at = CURRENT_TIMESTAMP
WHERE id IN (SELECT id FROM ordered);
`); err != nil {
		return fmt.Errorf("normalize positions after delete: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete transaction: %w", err)
	}

	return nil
}

func (s *SQLiteStore) ReorderTodos(ctx context.Context, orderedIDs []int64) error {
	if len(orderedIDs) == 0 {
		return errors.New("ordered ids are required")
	}

	var totalTodos int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM todos`).Scan(&totalTodos); err != nil {
		return fmt.Errorf("count todos before reorder: %w", err)
	}
	if totalTodos != len(orderedIDs) {
		return fmt.Errorf("ordered ids count must match todos count: got %d want %d", len(orderedIDs), totalTodos)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start reorder transaction: %w", err)
	}
	defer tx.Rollback()

	for index, id := range orderedIDs {
		if result, err := tx.ExecContext(ctx, `
UPDATE todos
SET position = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, index+1, id); err != nil {
			return fmt.Errorf("update todo %d position: %w", id, err)
		} else {
			affected, rowsErr := result.RowsAffected()
			if rowsErr != nil {
				return fmt.Errorf("resolve rows affected for todo %d: %w", id, rowsErr)
			}
			if affected == 0 {
				return fmt.Errorf("todo %d not found", id)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reorder transaction: %w", err)
	}

	return nil
}

func scanTodoRow(row interface{ Scan(dest ...any) error }) (Todo, error) {
	var todo Todo
	var completedInt int

	err := row.Scan(
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
			return Todo{}, errors.New("todo not found")
		}
		return Todo{}, fmt.Errorf("scan todo: %w", err)
	}

	todo.Completed = completedInt == 1
	return todo, nil
}
