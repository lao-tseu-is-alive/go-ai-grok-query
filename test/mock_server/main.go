package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	handler := http.NewServeMux()

	// Handler for OpenAI-compatible APIs (OpenAI, OpenRouter, XAI)
	handler.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"choices": [{"message": {"content": "Mock response for OpenAI-compatible API"}}]}`)
	})

	// Handler for Ollama
	handler.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"message": {"content": "Mock response for Ollama"}}`)
	})

	// Generic handler for Gemini, which includes the model name in the path
	handler.HandleFunc("/v1beta/models/", func(w http.ResponseWriter, r *http.Request) {
		// We only care that the path ends with ":generateContent"
		if strings.HasSuffix(r.URL.Path, ":generateContent") {
			fmt.Fprintln(w, `{"candidates": [{"content": {"parts": [{"text": "Mock response for Gemini"}]}}]}`)
		} else {
			http.NotFound(w, r)
		}
	})

	log.Println("Mock server starting on :8181")
	if err := http.ListenAndServe(":8181", handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
