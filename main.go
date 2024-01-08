package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"
    
	"github.com/gorilla/mux"
)

const baseURL = "http://bit.go/"

var store = make(map[string]string)

type Response struct {
	URL string `json:"url"`
}

// Handler for POST /api/shorten request 
func shortenHandler(w http.ResponseWriter, r *http.Request) {

    // Decode JSON body containing original URL
    var url map[string]string  
    json.NewDecoder(r.Body).Decode(&url) 

    // Generate short random URL code
    shortCode := generateShortUrl()  

    // Construct full shortened URL
    resUrl := baseURL + shortCode

    // Persist short code & original URL mapping 
    store[shortCode] = url["url"]

    // Prepare JSON response with shortened URL
    res := Response{URL: resUrl} 
    json.NewEncoder(w).Encode(res)

}

// Handler for GET /{code} redirect request
func redirectHandler(w http.ResponseWriter, r *http.Request) {
   
    // Extract short code from request URL 
    vars := mux.Vars(r)    
    code := vars["code"]

    // Retrieve original URL from store  
    if url, ok := store[code]; ok {
      
        // Redirect client to original URL
        http.Redirect(w, r, url, http.StatusTemporaryRedirect)

    } else {
      
        // Send 404 if short code not in store
        http.Error(w, "URL not found", http.StatusNotFound) 
    }
    
}

func generateShortUrl() string {
    // Generate a random short URL
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    const keyLength = 6

	rand.NewSource(time.Now().UnixNano())
	shortkey := make([]byte, keyLength)
	for i := range shortkey {
		shortkey[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortkey)
}

func main() {
	
	// Create Gorilla mux router
    router := mux.NewRouter()

    // Define API and redirect routes
    router.HandleFunc("/api/shorten", shortenHandler).Methods("POST")
    router.HandleFunc("/{code}", redirectHandler).Methods("GET")

    // Start HTTP server
    log.Println("Starting server on :8080")  
    http.ListenAndServe(":8080", router)
	
}

