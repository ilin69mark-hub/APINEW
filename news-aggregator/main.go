package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Config struct {
	Port string
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

func main() {
	config := Config{
		Port: getEnv("NEWS_AGGREGATOR_PORT", "8083"),
	}

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routes
	r.Get("/health", healthHandler)
	r.Get("/news", getNewsHandler)
	r.Get("/news/{id}", getNewsByIDHandler)

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

	log.Printf("News Aggregator starting on port %s", config.Port)
	<-done
	log.Println("Server stopped gracefully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func getNewsHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page == 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize == 0 {
		pageSize = 10
	}
	search := r.URL.Query().Get("search")

	// Generate mock news data
	news := generateMockNews(page, pageSize, search)

	// Calculate pagination
	total := len(news)
	totalPages := (total + pageSize - 1) / pageSize

	response := Response{
		Status: "success",
		Data:   news,
		Pagination: &Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getNewsByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid news ID", http.StatusBadRequest)
		return
	}

	// Find news by ID
	news := findNewsByID(id)
	if news == nil {
		http.Error(w, "News not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}

// Mock data generation functions
func generateMockNews(page, pageSize int, search string) []NewsItem {
	// Mock news data
	mockNews := []NewsItem{
		{
			ID:      1,
			Title:   "Новости технологий",
			Content: "В этом выпуске: последние обновления в мире технологий, новые релизы и тренды.",
			Date:    time.Now().Format("2006-01-02 15:04:05"),
		},
		{
			ID:      2,
			Title:   "Экономическая аналитика",
			Content: "Анализ текущей экономической ситуации и прогнозы на ближайшие месяцы.",
			Date:    time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			ID:      3,
			Title:   "Политические события",
			Content: "Обзор последних политических событий в стране и за рубежом.",
			Date:    time.Now().Add(-48 * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			ID:      4,
			Title:   "Спортивные новости",
			Content: "Результаты последних соревнований и интервью с известными спортсменами.",
			Date:    time.Now().Add(-12 * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			ID:      5,
			Title:   "Культура и искусство",
			Content: "Открытие новых выставок, премьеры фильмов и театральных постановок.",
			Date:    time.Now().Add(-36 * time.Hour).Format("2006-01-02 15:04:05"),
		},
	}

	// Apply search filter if provided
	if search != "" {
		var filtered []NewsItem
		for _, item := range mockNews {
			if containsIgnoreCase(item.Title, search) || containsIgnoreCase(item.Content, search) {
				filtered = append(filtered, item)
			}
		}
		mockNews = filtered
	}

	// Apply pagination
	start := (page - 1) * pageSize
	if start >= len(mockNews) {
		start = len(mockNews)
	}
	end := start + pageSize
	if end > len(mockNews) {
		end = len(mockNews)
	}

	return mockNews[start:end]
}

func findNewsByID(id int) *NewsItem {
	mockNews := []NewsItem{
		{
			ID:      1,
			Title:   "Новости технологий",
			Content: "В этом выпуске: последние обновления в мире технологий, новые релизы и тренды.",
			Date:    time.Now().Format("2006-01-02 15:04:05"),
		},
		{
			ID:      2,
			Title:   "Экономическая аналитика",
			Content: "Анализ текущей экономической ситуации и прогнозы на ближайшие месяцы.",
			Date:    time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			ID:      3,
			Title:   "Политические события",
			Content: "Обзор последних политических событий в стране и за рубежом.",
			Date:    time.Now().Add(-48 * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			ID:      4,
			Title:   "Спортивные новости",
			Content: "Результаты последних соревнований и интервью с известными спортсменами.",
			Date:    time.Now().Add(-12 * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			ID:      5,
			Title:   "Культура и искусство",
			Content: "Открытие новых выставок, премьеры фильмов и театральных постановок.",
			Date:    time.Now().Add(-36 * time.Hour).Format("2006-01-02 15:04:05"),
		},
	}

	for _, item := range mockNews {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

func containsIgnoreCase(s, substr string) bool {
	return contains(s, substr) || contains(s, toLower(substr))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func toLower(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result = append(result, c+('a'-'A'))
		} else {
			result = append(result, c)
		}
	}
	return string(result)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}