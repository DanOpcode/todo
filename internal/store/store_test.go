package store

import (
	"context"
	"testing"
)

func TestSQLiteStoreCreateAndListTodos(t *testing.T) {
	store, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	if _, err := store.CreateTodo(ctx, "First", "first item"); err != nil {
		t.Fatalf("create first todo: %v", err)
	}

	if _, err := store.CreateTodo(ctx, "Second", "second item"); err != nil {
		t.Fatalf("create second todo: %v", err)
	}

	todos, err := store.ListTodos(ctx)
	if err != nil {
		t.Fatalf("list todos: %v", err)
	}

	if len(todos) != 2 {
		t.Fatalf("unexpected todo count: got %d want 2", len(todos))
	}

	if todos[0].Position != 1 || todos[1].Position != 2 {
		t.Fatalf("unexpected positions: got [%d, %d]", todos[0].Position, todos[1].Position)
	}
}
