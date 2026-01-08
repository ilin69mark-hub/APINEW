package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type Config struct {
	Port string
}

type CheckRequest struct {
	Text string `json:"text"`
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

type CensorService struct {
	bannedWords map[string]bool
	mutex       sync.RWMutex
}

func main() {
	config := Config{
		Port: getEnv("CENSOR_SERVICE_PORT", "8082"),
	}

	censorService := NewCensorService()

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
	r.Post("/check", censorService.checkHandler)

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

	log.Printf("Censor Service starting on port %s", config.Port)
	<-done
	log.Println("Server stopped gracefully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewCensorService() *CensorService {
	cs := &CensorService{
		bannedWords: make(map[string]bool),
	}
	
	// Initialize banned words
	bannedWords := []string{
		"qwerty", "йцукен", "zxvbnm",
		// Additional banned words can be added here
	}
	
	for _, word := range bannedWords {
		cs.bannedWords[strings.ToLower(word)] = true
	}
	
	return cs
}

func (cs *CensorService) IsBanned(text string) bool {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	
	lowerText := strings.ToLower(text)
	
	// Check for banned words in the text
	for word := range cs.bannedWords {
		if strings.Contains(lowerText, word) {
			return true
		}
	}
	
	return false
}

func (cs *CensorService) AddBannedWord(word string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	cs.bannedWords[strings.ToLower(strings.TrimSpace(word))] = true
}

func (cs *CensorService) RemoveBannedWord(word string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	delete(cs.bannedWords, strings.ToLower(strings.TrimSpace(word)))
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func (cs *CensorService) checkHandler(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if cs.IsBanned(req.Text) {
		http.Error(w, "Text contains prohibited content", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   map[string]string{"message": "Text is clean"},
	})
}