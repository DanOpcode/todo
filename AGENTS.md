# Project Init Notes

## Stack
- Backend: Go (`net/http`, standard library templates)
- Frontend: server-rendered HTML (single page for now)
- Data: hardcoded todos (no persistent storage yet)

## Run
- Start app: `go run .`
- Open: `http://localhost:8080`

## Test
- Run all tests: `go test ./...`

## Current Structure
- `main.go`: HTTP server, inlined HTML template, sample todo data
- `README.md`: one-line project intro

## Next Milestones
- Move HTML into dedicated template files
- Add persistent storage (SQLite)
- Add JSON API for todo CRUD operations
- Add interactive frontend behavior
