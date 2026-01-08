package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	Port string
	DBPath string
}

type Response struct {
	Status     string      `json:"status"`
	Data       interface{} `json:"data,omitempty"`
	Error      string      `json:"error,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	Total     int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type Comment struct {
	ID        int            `json:"id"`
	NewsID    int            `json:"news_id"`
	ParentID  *int           `json:"parent_id,omitempty"`
	Text      string         `json:"text"`
	CreatedAt string         `json:"created_at"`
}

type CommentRequest struct {
	NewsID   int    `json:"news_id"`
	ParentID *int   `json:"parent_id,omitempty"`
	Text     string `json:"text"`
}

func main() {
	config := Config{
		Port:   getEnv("COMMENT_SERVICE_PORT", "8081"),
		DBPath: getEnv("COMMENT_DB_PATH", "./comments.db"),
	}

	db, err := initDB(config.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(requestIDMiddleware)
	r.Use(loggerMiddleware)

	// Routes
	r.Get("/health", healthHandler)
	r.Get("/comments", getCommentsHandler(db))
	r.Post("/comments", createCommentHandler(db))
	r.Delete("/comments/{id}", deleteCommentHandler(db))

	// Graceful shutdown
	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	log.Printf("Comment Service starting on port %s", config.Port)
	<-done
	log.Println("Server stopped gracefully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s %s", 
			r.Context().Value("request_id"), 
			r.Method, 
			r.URL.Path, 
			time.Since(start))
	})
}

func initDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create comments table
	query := `
	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		news_id INTEGER NOT NULL,
		parent_id INTEGER,
		text TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (parent_id) REFERENCES comments (id)
	);
	CREATE INDEX IF NOT EXISTS idx_news_id ON comments(news_id);
	CREATE INDEX IF NOT EXISTS idx_parent_id ON comments(parent_id);
	`
	
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func getCommentsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newsIDStr := r.URL.Query().Get("news_id")
		if newsIDStr == "" {
			http.Error(w, "news_id parameter is required", http.StatusBadRequest)
			return
		}

		newsID, err := strconv.Atoi(newsIDStr)
		if err != nil {
			http.Error(w, "Invalid news_id parameter", http.StatusBadRequest)
			return
		}

		// Query comments for the news item
		query := "SELECT id, news_id, parent_id, text, created_at FROM comments WHERE news_id = ? ORDER BY created_at ASC"
		rows, err := db.Query(query, newsID)
		if err != nil {
			http.Error(w, "Failed to fetch comments", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var comments []Comment
		for rows.Next() {
			var comment Comment
			var parentID sql.NullInt64
			err := rows.Scan(&comment.ID, &comment.NewsID, &parentID, &comment.Text, &comment.CreatedAt)
			if err != nil {
				http.Error(w, "Failed to scan comment", http.StatusInternalServerError)
				return
			}
			
			if parentID.Valid {
				pid := int(parentID.Int64)
				comment.ParentID = &pid
			}
			
			comments = append(comments, comment)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Status: "success",
			Data:   comments,
		})
	}
}

func createCommentHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CommentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate input
		if req.Text == "" {
			http.Error(w, "Comment text is required", http.StatusBadRequest)
			return
		}
		if req.NewsID <= 0 {
			http.Error(w, "Valid news ID is required", http.StatusBadRequest)
			return
		}
		
		// Check if parent_id exists if provided
		if req.ParentID != nil {
			var exists int
			err := db.QueryRow("SELECT 1 FROM comments WHERE id = ?", *req.ParentID).Scan(&exists)
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, "Parent comment does not exist", http.StatusBadRequest)
					return
				}
				http.Error(w, "Failed to validate parent comment", http.StatusInternalServerError)
				return
			}
		}

		// Insert comment
		var parentID *int
		if req.ParentID != nil && *req.ParentID > 0 {
			parentID = req.ParentID
		}
		
		query := "INSERT INTO comments (news_id, parent_id, text) VALUES (?, ?, ?)"
		result, err := db.Exec(query, req.NewsID, parentID, req.Text)
		if err != nil {
			http.Error(w, "Failed to save comment", http.StatusInternalServerError)
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Failed to get inserted comment ID", http.StatusInternalServerError)
			return
		}

		// Get the inserted comment
		var comment Comment
		var parentId sql.NullInt64
		err = db.QueryRow("SELECT id, news_id, parent_id, text, created_at FROM comments WHERE id = ?", id).Scan(
			&comment.ID, &comment.NewsID, &parentId, &comment.Text, &comment.CreatedAt)
		if err != nil {
			http.Error(w, "Failed to fetch inserted comment", http.StatusInternalServerError)
			return
		}
		
		if parentId.Valid {
			pid := int(parentId.Int64)
			comment.ParentID = &pid
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Status: "success",
			Data:   comment,
		})
	}
}

func deleteCommentHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid comment ID", http.StatusBadRequest)
			return
		}

		// Check if comment exists
		var exists int
		err = db.QueryRow("SELECT 1 FROM comments WHERE id = ?", id).Scan(&exists)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Comment not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to check comment existence", http.StatusInternalServerError)
			return
		}

		// Delete comment
		_, err = db.Exec("DELETE FROM comments WHERE id = ?", id)
		if err != nil {
			http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Status: "success",
			Data:   map[string]string{"message": "Comment deleted successfully"},
		})
	}
}