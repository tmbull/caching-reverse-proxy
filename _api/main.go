package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var mu sync.RWMutex
var documents = make(map[string]map[string]interface{}, 10)

func main() {
	username := os.Getenv("USERNAME")
	if username == "" {
		username = "user"
	}
	password := os.Getenv("PASSWORD")
	if password == "" {
		password = "password"
	}
	r := mux.NewRouter()
	r.HandleFunc("/db/documents", createDocument).Methods("POST")
	r.HandleFunc("/db/documents/{id}", deleteDocument).Methods("DELETE")
	r.HandleFunc("/db/documents/{id}", getDocument).Methods("GET")
	r.HandleFunc("/db/query", queryDocuments)
	r.Use(makeAuthMiddleware(username, password))

	log.Fatal(http.ListenAndServe(":8080", r))
}

func makeAuthMiddleware(username string, password string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if user, pass, ok := r.BasicAuth(); ok && user == username && pass == password {
			log.Printf("Authenticated user %s\n", user)
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	})}
}

func createDocument(writer http.ResponseWriter, request *http.Request) {
	key := uuid.NewString()
	var document map[string]interface{}
	err := json.NewDecoder(request.Body).Decode(&document)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	mu.Lock()
	documents[key] = document
	mu.Unlock()

	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write([]byte(key))
	if err != nil {
		log.Printf("Failed to write key to response: %v", err)
		return
	}
}

func deleteDocument(_ http.ResponseWriter, request *http.Request) {
	key := mux.Vars(request)["id"]
	mu.Lock()
	delete(documents, key)
	mu.Unlock()
}

func getDocument(writer http.ResponseWriter, request *http.Request) {
	key := mux.Vars(request)["id"]

	mu.RLock()
	result, ok := documents[key]
	mu.RUnlock()

	if ok {
		err := json.NewEncoder(writer).Encode(result)
		if err != nil {
			log.Printf("JSON encoding failed: %v", err)
		}
	} else {
		writer.WriteHeader(http.StatusNotFound)
	}
}

func queryDocuments(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	result := make([]map[string]interface{}, 10)

	mu.RLock()
	for _, doc := range documents {
		found := true
		for field, val := range query {
			// Only supporting single value query params for now
			if val[0] != doc[field] {
				found = false
				break
			}
		}

		if found {
			result = append(result, doc)
		}
	}
	mu.RUnlock()
	err := json.NewEncoder(writer).Encode(result)
	if err != nil {
		log.Printf("JSON encoding failed: %v", err)
	}
}