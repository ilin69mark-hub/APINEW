package main

import (
	"context"
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
)

type Config struct {
	Port             string
	CommentServiceURL string
	CensorServiceURL  string
	NewsAggregatorURL string
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

type NewsItem struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Date    string `json:"date"`
}

type Comment struct {
	ID       int    `json:"id"`
	NewsID   int    `json:"news_id"`
	ParentID *int   `json:"parent_id,omitempty"`
	Text     string `json:"text"`
	CreatedAt string `json:"created_at"`
}

type CommentRequest struct {
	NewsID   int    `json:"news_id"`
	ParentID *int   `json:"parent_id,omitempty"`
	Text     string `json:"text"`
}

func main() {
	config := Config{
		Port:             getEnv("API_GATEWAY_PORT", "8080"),
		CommentServiceURL: getEnv("COMMENT_SERVICE_URL", "http://comment-service:8081"),
		CensorServiceURL:  getEnv("CENSOR_SERVICE_URL", "http://censor-service:8082"),
		NewsAggregatorURL: getEnv("NEWS_AGGREGATOR_URL", "http://news-aggregator:8083"),
	}

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(requestIDMiddleware)
	r.Use(loggerMiddleware)
	r.Use(timeoutMiddleware(30 * time.Second))

	// Routes
	r.Get("/health", healthHandler)
	r.Get("/news", getNewsHandler(config))
	r.Get("/news/{id}", getNewsByIDHandler(config))
	r.Post("/comment", createCommentHandler(config))

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

	log.Printf("API Gateway starting on port %s", config.Port)
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

func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Request timeout")
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func getNewsHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page == 0 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		if pageSize == 0 {
			pageSize = 10
		}
		search := r.URL.Query().Get("search")

		// Call News Aggregator service
		url := fmt.Sprintf("%s/news?page=%d&page_size=%d", config.NewsAggregatorURL, page, pageSize)
		if search != "" {
			url += "&search=" + search
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			http.Error(w, "Failed to fetch news", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var newsResponse Response
		if err := json.NewDecoder(resp.Body).Decode(&newsResponse); err != nil {
			http.Error(w, "Failed to decode news response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newsResponse)
	}
}

func getNewsByIDHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newsID := chi.URLParam(r, "id")
		newsIDInt, err := strconv.Atoi(newsID)
		if err != nil {
			http.Error(w, "Invalid news ID", http.StatusBadRequest)
			return
		}

		// Fetch news from News Aggregator
		newsURL := fmt.Sprintf("%s/news/%s", config.NewsAggregatorURL, newsID)
		client := &http.Client{Timeout: 10 * time.Second}
		newsResp, err := client.Get(newsURL)
		if err != nil {
			http.Error(w, "Failed to fetch news", http.StatusInternalServerError)
			return
		}
		defer newsResp.Body.Close()

		var newsItem NewsItem
		if err := json.NewDecoder(newsResp.Body).Decode(&newsItem); err != nil {
			http.Error(w, "Failed to decode news response", http.StatusInternalServerError)
			return
		}

		// Fetch comments for this news
		commentsURL := fmt.Sprintf("%s/comments?news_id=%d", config.CommentServiceURL, newsIDInt)
		commentsResp, err := client.Get(commentsURL)
		if err != nil {
			http.Error(w, "Failed to fetch comments", http.StatusInternalServerError)
			return
		}
		defer commentsResp.Body.Close()

		var commentsResponse Response
		if err := json.NewDecoder(commentsResp.Body).Decode(&commentsResponse); err != nil {
			http.Error(w, "Failed to decode comments response", http.StatusInternalServerError)
			return
		}

		// Combine news and comments
		result := map[string]interface{}{
			"news":      newsItem,
			"comments":  commentsResponse.Data,
			"request_id": r.Context().Value("request_id"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Status: "success", Data: result})
	}
}

func createCommentHandler(config Config) http.HandlerFunc {
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

		// Check with Censor Service
		censorURL := config.CensorServiceURL + "/check"
		censorPayload := map[string]string{"text": req.Text}
		censorPayloadBytes, _ := json.Marshal(censorPayload)

		client := &http.Client{Timeout: 10 * time.Second}
		censorReq, _ := http.NewRequest("POST", censorURL, strings.NewReader(string(censorPayloadBytes)))
		censorReq.Header.Set("Content-Type", "application/json")
		censorReq.Header.Set("X-Request-ID", r.Context().Value("request_id").(string))

		censorResp, err := client.Do(censorReq)
		if err != nil {
			http.Error(w, "Failed to check comment with censor service", http.StatusInternalServerError)
			return
		}
		defer censorResp.Body.Close()

		if censorResp.StatusCode != http.StatusOK {
			http.Error(w, "Comment contains prohibited content", http.StatusBadRequest)
			return
		}

		// Forward to Comment Service
		commentURL := config.CommentServiceURL + "/comments"
		commentPayloadBytes, _ := json.Marshal(req)

		commentReq, _ := http.NewRequest("POST", commentURL, strings.NewReader(string(commentPayloadBytes)))
		commentReq.Header.Set("Content-Type", "application/json")
		commentReq.Header.Set("X-Request-ID", r.Context().Value("request_id").(string))

		commentResp, err := client.Do(commentReq)
		if err != nil {
			http.Error(w, "Failed to save comment", http.StatusInternalServerError)
			return
		}
		defer commentResp.Body.Close()

		if commentResp.StatusCode != http.StatusOK {
			http.Error(w, "Failed to save comment", http.StatusInternalServerError)
			return
		}

		var commentResponse Response
		if err := json.NewDecoder(commentResp.Body).Decode(&commentResponse); err != nil {
			http.Error(w, "Failed to decode comment response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(commentResponse)
	}
}