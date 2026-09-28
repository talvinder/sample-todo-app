package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

const cacheKey = "todos:all"

type Todo struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type server struct {
	db  *sql.DB
	rdb *redis.Client
}

func main() {
	db, err := sql.Open("mysql", env("MYSQL_DSN", "root:root@tcp(localhost:3306)/todos?parseTime=true"))
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; db.Ping() != nil; i++ {
		if i == 30 {
			log.Fatal("mysql not reachable")
		}
		time.Sleep(time.Second)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS todos (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		done BOOLEAN NOT NULL DEFAULT FALSE)`); err != nil {
		log.Fatal(err)
	}

	s := &server{db: db, rdb: redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "localhost:6379")})}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/todos", s.list)
	mux.HandleFunc("POST /api/todos", s.create)
	mux.HandleFunc("PATCH /api/todos/{id}", s.toggle)
	mux.HandleFunc("DELETE /api/todos/{id}", s.remove)

	log.Println("backend listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// list is cache-aside: serve from Redis if present, otherwise read MySQL and fill the cache.
func (s *server) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if cached, err := s.rdb.Get(ctx, cacheKey).Bytes(); err == nil {
		w.Header().Set("X-Cache", "HIT")
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}

	rows, err := s.db.QueryContext(ctx, "SELECT id, title, done FROM todos ORDER BY id")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	todos := []Todo{}
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Done); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		todos = append(todos, t)
	}

	body, _ := json.Marshal(todos)
	s.rdb.Set(ctx, cacheKey, body, time.Minute)
	w.Header().Set("X-Cache", "MISS")
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (s *server) create(w http.ResponseWriter, r *http.Request) {
	var t Todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil || t.Title == "" {
		http.Error(w, "title required", 400)
		return
	}
	res, err := s.db.ExecContext(r.Context(), "INSERT INTO todos (title) VALUES (?)", t.Title)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	t.ID, _ = res.LastInsertId()
	s.invalidate(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(t)
}

func (s *server) toggle(w http.ResponseWriter, r *http.Request) {
	s.exec(w, r, "UPDATE todos SET done = NOT done WHERE id = ?")
}

func (s *server) remove(w http.ResponseWriter, r *http.Request) {
	s.exec(w, r, "DELETE FROM todos WHERE id = ?")
}

func (s *server) exec(w http.ResponseWriter, r *http.Request, query string) {
	res, err := s.db.ExecContext(r.Context(), query, r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "not found", 404)
		return
	}
	s.invalidate(r.Context())
	w.WriteHeader(204)
}

func (s *server) invalidate(ctx context.Context) { s.rdb.Del(ctx, cacheKey) }

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
