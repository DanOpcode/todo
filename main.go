package main

import (
	"log"
	"net/http"
	"text/template"
)

type Todo struct {
	Title       string
	Description string
	Done        bool
}

func main() {
	tmpl := template.Must(template.ParseFiles("templates/todos.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		todos := []Todo{
			{Title: "Learn Go", Description: "Study Go basics", Done: true},
			{Title: "Build todo app", Description: "Create a todo web app", Done: false},
			{Title: "Deploy app", Description: "Deploy to production", Done: false},
		}
		if err := tmpl.Execute(w, todos); err != nil {
			http.Error(w, "failed to render todos", http.StatusInternalServerError)
		}
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
