package store

import (
	"context"
	"testing"
)

func TestSQLiteStoreCRUDAndReorder(t *testing.T) {
	ctx := context.Background()

	store, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	first, err := store.CreateTodo(ctx, "First", "one")
	if err != nil {
		t.Fatalf("create first todo: %v", err)
	}

	second, err := store.CreateTodo(ctx, "Second", "two")
	if err != nil {
		t.Fatalf("create second todo: %v", err)
	}

	if _, err := store.SetCompleted(ctx, second.ID, true); err != nil {
		t.Fatalf("set completion: %v", err)
	}

	if _, err := store.UpdateTodo(ctx, first.ID, "First updated", "one updated"); err != nil {
		t.Fatalf("update todo: %v", err)
	}

	if err := store.ReorderTodos(ctx, []int64{second.ID, first.ID}); err != nil {
		t.Fatalf("reorder todos: %v", err)
	}

	todos, err := store.ListTodos(ctx)
	if err != nil {
		t.Fatalf("list todos: %v", err)
	}

	if len(todos) != 2 {
		t.Fatalf("unexpected todo count: got %d want 2", len(todos))
	}

	if todos[0].ID != second.ID || todos[0].Position != 1 || !todos[0].Completed {
		t.Fatalf("unexpected reordered first todo: %+v", todos[0])
	}

	if err := store.DeleteTodo(ctx, first.ID); err != nil {
		t.Fatalf("delete todo: %v", err)
	}

	todos, err = store.ListTodos(ctx)
	if err != nil {
		t.Fatalf("list todos after delete: %v", err)
	}

	if len(todos) != 1 || todos[0].Position != 1 {
		t.Fatalf("unexpected remaining todos: %+v", todos)
	}
}
