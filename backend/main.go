package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/blog?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to database!")

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rows, err := db.Query("SELECT id, title, content FROM posts")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			var postsdb.Query("SELECT id, title, content FROM posts") []map[string]string
			for rows.Next() {
				var id, title, content string
				rows.Scan(&id, &title, &content)
				posts = append(posts, map[string]string{"id": id, "title": title, "content": content})
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"posts": %v}`, posts)

		case http.MethodPost:
			title := r.FormValue("title")
			content := r.FormValue("content")
			if title == "" || content == "" {
				http.Error(w, "title and content are required", http.StatusBadRequest)
				return
			}
			err := db.QueryRow("INSERT INTO posts (title, content) VALUES ($1, $2) RETURNING id", title, content).Scan(&id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id": "%s"}`, id)
		}
	})

	http.HandleFunc("/api/posts/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			id := r.URL.Path[len("/api/posts/"):]
			row := db.QueryRow("SELECT id, title, content FROM posts WHERE id = $1", id)
			var id2, title, content string
			if err := row.Scan(&id2, &title, &content); err != nil {
				http.Error(w, "post not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id": "%s", "title": "%s", "content": "%s"}`, id2, title, content)

		case http.MethodPut:
			id := r.URL.Path[len("/api/posts/"):]
			title := r.FormValue("title")
			content := r.FormValue("content")
			_, err := db.Exec("UPDATE posts SET title = $1, content = $2 WHERE id = $3", title, content, id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		case http.MethodDelete:
			id := r.URL.Path[len("/api/posts/"):]
			_, err := db.Exec("DELETE FROM posts WHERE id = $1", id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

var id string

func initDB(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS posts (id SERIAL PRIMARY KEY, title TEXT NOT NULL, content TEXT NOT NULL)`)
	return err
}