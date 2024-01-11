package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// Test generateShortUrl function
func TestGenerateShortUrl(t *testing.T) {
	shortUrl := generateShortUrl()
	if len(shortUrl) != 6 {
		t.Errorf("Short URL length should be 6, got %d", len(shortUrl))
	}
	if !strings.ContainsAny(shortUrl, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") {
		t.Errorf("Short URL should contain only alphanumeric characters")
	}
}

// Test shortenHandler function
func TestShortenHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(`{"url": "https://example.com"}`))
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/shorten", shortenHandler).Methods("POST")
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var res Response
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(res.URL, baseURL) {
		t.Errorf("Shortened URL should start with %s", baseURL)
	}
	if _, ok := store[strings.TrimPrefix(res.URL, baseURL)]; !ok {
		t.Errorf("Short code not found in store")
	}
}

// Test redirectHandler function
func TestRedirectHandler(t *testing.T) {
	shortCode := "test123"
	originalURL := "https://example.com"
	store[shortCode] = originalURL

	req, err := http.NewRequest("GET", "/"+shortCode, nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/{code}", redirectHandler).Methods("GET")
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusTemporaryRedirect {
		t.Errorf("Expected status code %d, got %d", http.StatusTemporaryRedirect, status)
	}

	if location := rr.Header().Get("Location"); location != originalURL {
		t.Errorf("Expected redirect location %s, got %s", originalURL, location)
	}
}
