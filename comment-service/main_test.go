package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("could not unmarshal response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("handler returned unexpected body: got %v want %v",
			response.Status, "ok")
	}
}

func TestInitDB(t *testing.T) {
	db, err := initDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Test that tables were created
	_, err = db.Exec("SELECT 1 FROM comments LIMIT 1")
	if err != nil {
		t.Errorf("comments table was not created properly: %v", err)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value("request_id")
		if requestID == nil {
			t.Error("request_id not found in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := requestIDMiddleware(nextHandler)
	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	
	middleware.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			rr.Code, http.StatusOK)
	}
}