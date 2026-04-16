package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"todo/internal/store"
)

type apiServer struct {
	store *store.SQLiteStore
}

type createTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type updateTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   *bool  `json:"completed"`
}

type reorderTodosRequest struct {
	OrderedIDs []int64 `json:"orderedIds"`
}

func main() {
	dbPath := os.Getenv("TODO_DB_PATH")
	if dbPath == "" {
		dbPath = "todo.db"
	}

	dbStore, err := store.OpenSQLite(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize sqlite store: %v", err)
	}
	defer dbStore.Close()

	server := &apiServer{store: dbStore}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/todos", server.handleTodos)
	mux.HandleFunc("/api/todos/reorder", server.handleReorderTodos)
	mux.HandleFunc("/api/todos/", server.handleTodoByID)

	if hasBuiltWeb() {
		fileServer := http.FileServer(http.Dir("web/dist"))
		mux.Handle("/", fileServer)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "web build not found. run: cd web && npm install && npm run build", http.StatusNotFound)
		})
	}

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}

func (s *apiServer) handleTodos(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		todos, err := s.store.ListTodos(ctx)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, todos)
	case http.MethodPost:
		var input createTodoRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		todo, err := s.store.CreateTodo(ctx, input.Title, input.Description)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, todo)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *apiServer) handleTodoByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id, err := parseTodoID(r.URL.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid todo id")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var input updateTodoRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid json body")
			return
		}

		todo, err := s.store.UpdateTodo(ctx, id, input.Title, input.Description)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		if input.Completed != nil {
			todo, err = s.store.SetCompleted(ctx, id, *input.Completed)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		writeJSON(w, http.StatusOK, todo)
	case http.MethodDelete:
		if err := s.store.DeleteTodo(ctx, id); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *apiServer) handleReorderTodos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var input reorderTodosRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := s.store.ReorderTodos(ctx, input.OrderedIDs); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	todos, err := s.store.ListTodos(ctx)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, todos)
}

func parseTodoID(path string) (int64, error) {
	idPart := strings.TrimPrefix(path, "/api/todos/")
	idPart = strings.TrimSpace(idPart)
	if idPart == "" || strings.Contains(idPart, "/") {
		return 0, errors.New("invalid todo id")
	}

	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		return 0, err
	}

	if id <= 0 {
		return 0, errors.New("todo id must be positive")
	}

	return id, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func hasBuiltWeb() bool {
	info, err := os.Stat(filepath.Join("web", "dist", "index.html"))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
