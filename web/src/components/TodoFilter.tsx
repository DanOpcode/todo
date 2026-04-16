import { useMemo, useState } from "react";

type Todo = {
  title: string;
  description: string;
  done: boolean;
};

const initialTodos: Todo[] = [
  { title: "Learn Go", description: "Study Go basics", done: true },
  { title: "Build todo app", description: "Create a todo web app", done: false },
  { title: "Deploy app", description: "Deploy to production", done: false }
];

export default function TodoFilter() {
  const [showDone, setShowDone] = useState(true);
  const [showPending, setShowPending] = useState(true);

  const visibleTodos = useMemo(() => {
    return initialTodos.filter((todo) => {
      if (todo.done && !showDone) return false;
      if (!todo.done && !showPending) return false;
      return true;
    });
  }, [showDone, showPending]);

  return (
    <section>
      <h2>Interactive Todo Filter</h2>
      <div style={{ display: "flex", gap: "1rem", marginBottom: "1rem" }}>
        <label>
          <input
            type="checkbox"
            checked={showDone}
            onChange={(event) => setShowDone(event.target.checked)}
          />{" "}
          Show done
        </label>
        <label>
          <input
            type="checkbox"
            checked={showPending}
            onChange={(event) => setShowPending(event.target.checked)}
          />{" "}
          Show pending
        </label>
      </div>

      <ul>
        {visibleTodos.map((todo) => (
          <li key={todo.title}>
            <strong>{todo.title}</strong>: {todo.description} ({todo.done ? "Done" : "Pending"})
          </li>
        ))}
      </ul>
    </section>
  );
}
