package main

import (
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

func TestGetNewsHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/news?page=1&page_size=10", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getNewsHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("could not unmarshal response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("handler returned unexpected status: got %v want %v",
			response.Status, "success")
	}
}

func TestGetNewsByIDHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/news/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getNewsByIDHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var news NewsItem
	if err := json.Unmarshal(rr.Body.Bytes(), &news); err != nil {
		t.Errorf("could not unmarshal response: %v", err)
	}

	if news.ID != 1 {
		t.Errorf("handler returned unexpected news ID: got %v want %v",
			news.ID, 1)
	}
}