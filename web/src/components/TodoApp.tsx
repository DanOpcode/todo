import { useEffect, useMemo, useState } from "react";

type Todo = {
  id: number;
  title: string;
  description: string;
  completed: boolean;
  position: number;
};

type TodoDraft = {
  title: string;
  description: string;
};

const emptyDraft: TodoDraft = {
  title: "",
  description: "",
};

async function readJSON<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: "Request failed" }));
    throw new Error(payload.error ?? "Request failed");
  }
  return response.json() as Promise<T>;
}

export default function TodoApp() {
  const [todos, setTodos] = useState<Todo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [draft, setDraft] = useState<TodoDraft>(emptyDraft);
  const [editingId, setEditingId] = useState<number | null>(null);

  const sortedTodos = useMemo(() => {
    return [...todos].sort((a, b) => a.position - b.position);
  }, [todos]);

  useEffect(() => {
    void loadTodos();
  }, []);

  async function loadTodos() {
    try {
      setLoading(true);
      const response = await fetch("/api/todos");
      const payload = await readJSON<Todo[]>(response);
      setTodos(payload);
      setError(null);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Failed to load todos");
    } finally {
      setLoading(false);
    }
  }

  async function saveTodo() {
    if (!draft.title.trim()) {
      setError("Title is required");
      return;
    }

    try {
      setError(null);
      if (editingId === null) {
        const created = await readJSON<Todo>(
          await fetch("/api/todos", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(draft),
          })
        );
        setTodos((current) => [...current, created]);
      } else {
        const existingTodo = sortedTodos.find((todo) => todo.id === editingId);
        const updated = await readJSON<Todo>(
          await fetch(`/api/todos/${editingId}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              ...draft,
              completed: existingTodo?.completed ?? false,
            }),
          })
        );
        setTodos((current) => current.map((todo) => (todo.id === updated.id ? updated : todo)));
      }

      setDraft(emptyDraft);
      setEditingId(null);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Failed to save todo");
    }
  }

  async function toggleCompleted(todo: Todo) {
    try {
      const updated = await readJSON<Todo>(
        await fetch(`/api/todos/${todo.id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            title: todo.title,
            description: todo.description,
            completed: !todo.completed,
          }),
        })
      );
      setTodos((current) => current.map((item) => (item.id === updated.id ? updated : item)));
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Failed to toggle todo");
    }
  }

  async function removeTodo(id: number) {
    try {
      const response = await fetch(`/api/todos/${id}`, { method: "DELETE" });
      if (!response.ok) {
        const payload = await response.json().catch(() => ({ error: "Delete failed" }));
        throw new Error(payload.error ?? "Delete failed");
      }
      setTodos((current) => current.filter((todo) => todo.id !== id));
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Failed to delete todo");
    }
  }

  async function reorderTodo(id: number, direction: "up" | "down") {
    const index = sortedTodos.findIndex((todo) => todo.id === id);
    if (index < 0) return;

    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= sortedTodos.length) return;

    const reordered = [...sortedTodos];
    [reordered[index], reordered[targetIndex]] = [reordered[targetIndex], reordered[index]];

    const orderedIds = reordered.map((todo) => todo.id);
    try {
      const updatedTodos = await readJSON<Todo[]>(
        await fetch("/api/todos/reorder", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ orderedIds }),
        })
      );
      setTodos(updatedTodos);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Failed to reorder todos");
    }
  }

  function beginEdit(todo: Todo) {
    setEditingId(todo.id);
    setDraft({ title: todo.title, description: todo.description });
  }

  function cancelEdit() {
    setEditingId(null);
    setDraft(emptyDraft);
  }

  return (
    <main>
      <section className="panel">
        <h1>Todo List</h1>
        <p className="subtitle">Astro page + React island + Go API</p>

        <div className="form-grid">
          <label>
            Title
            <input
              value={draft.title}
              onChange={(event) => setDraft({ ...draft, title: event.target.value })}
              placeholder="Write docs"
            />
          </label>
          <label>
            Description
            <textarea
              value={draft.description}
              onChange={(event) => setDraft({ ...draft, description: event.target.value })}
              placeholder="Include API examples"
              rows={3}
            />
          </label>
          <div className="form-actions">
            <button onClick={saveTodo}>{editingId === null ? "Add Todo" : "Save Changes"}</button>
            {editingId !== null && (
              <button className="secondary" onClick={cancelEdit}>
                Cancel
              </button>
            )}
          </div>
        </div>

        {error && <p className="error">{error}</p>}

        {loading ? (
          <p>Loading todos...</p>
        ) : (
          <ul className="todo-list">
            {sortedTodos.map((todo, index) => (
              <li key={todo.id} className={todo.completed ? "done" : "pending"}>
                <div>
                  <strong>{todo.title}</strong>
                  <p>{todo.description || "No description"}</p>
                </div>
                <div className="item-actions">
                  <button className="secondary" onClick={() => toggleCompleted(todo)}>
                    {todo.completed ? "Mark Pending" : "Mark Done"}
                  </button>
                  <button className="secondary" onClick={() => beginEdit(todo)}>
                    Edit
                  </button>
                  <button className="secondary" onClick={() => reorderTodo(todo.id, "up")} disabled={index === 0}>
                    Up
                  </button>
                  <button
                    className="secondary"
                    onClick={() => reorderTodo(todo.id, "down")}
                    disabled={index === sortedTodos.length - 1}
                  >
                    Down
                  </button>
                  <button className="danger" onClick={() => removeTodo(todo.id)}>
                    Delete
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
