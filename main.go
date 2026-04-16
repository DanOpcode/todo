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
	tmpl := template.Must(template.New("todos").Parse(`
<!DOCTYPE html>
<html>
<head>
	<title>Todo App</title>
</head>
<body>
	<h1>Todos</h1>
	<ul>
		{{range .}}
		<li>{{.Title}}: {{.Description}} {{if .Done}}(Done){{else}}(Pending){{end}}</li>
		{{end}}
	</ul>
</body>
</html>
`))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		todos := []Todo{
			{Title: "Learn Go", Description: "Study Go basics", Done: true},
			{Title: "Build todo app", Description: "Create a todo web app", Done: false},
			{Title: "Deploy app", Description: "Deploy to production", Done: false},
		}
		tmpl.Execute(w, todos)
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
