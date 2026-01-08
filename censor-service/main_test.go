package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCensorService(t *testing.T) {
	cs := NewCensorService()

	// Test clean text
	if cs.IsBanned("This is a clean text") {
		t.Error("Expected clean text to pass")
	}

	// Test banned text
	if !cs.IsBanned("This contains qwerty") {
		t.Error("Expected banned text to fail")
	}

	// Test case insensitive
	if !cs.IsBanned("This contains QWERTY") {
		t.Error("Expected banned text (uppercase) to fail")
	}

	// Test Cyrillic
	if !cs.IsBanned("This contains йцукен") {
		t.Error("Expected banned Cyrillic text to fail")
	}
}

func TestCheckHandler(t *testing.T) {
	cs := NewCensorService()

	// Test clean text
	reqBody := `{"text": "This is a clean text"}`
	req, _ := http.NewRequest("POST", "/check", strings.NewReader(reqBody))
	rr := httptest.NewRecorder()
	
	cs.checkHandler(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("Expected clean text to pass with status 200, got %d", rr.Code)
	}

	// Test banned text
	reqBody = `{"text": "This contains qwerty"}`
	req, _ = http.NewRequest("POST", "/check", strings.NewReader(reqBody))
	rr = httptest.NewRecorder()
	
	cs.checkHandler(rr, req)
	
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected banned text to fail with status 400, got %d", rr.Code)
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