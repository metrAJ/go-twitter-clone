package twitter_handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"twitter/internal/models"
)

type Service interface {
	CreateTweet(ctx context.Context, content string) error
	GetTweets(ctx context.Context, limit int, cursor string) ([]*models.Message, string, error)
	GetStream() (<-chan []byte, string, error)
	CloseStream(queueName string)
}

type FeedHandler struct {
	svc Service
}

func NewFeedHandler(svc Service) *FeedHandler {
	return &FeedHandler{svc: svc}
}

func (h *FeedHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error": "Invalid JSON format"}`, http.StatusBadRequest)
		return
	}
	if payload.Content == "" {
		http.Error(w, `{"error": "Content cannot be empty"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.CreateTweet(r.Context(), payload.Content); err != nil {
		http.Error(w, `{"error": "Failed to create tweet"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "accepted"}`))
}

func (h *FeedHandler) GetRecentMessages(w http.ResponseWriter, r *http.Request) {
	cursorStr := r.URL.Query().Get("cursor")
	limit := 20
	messages, nextCursor, err := h.svc.GetTweets(r.Context(), limit, cursorStr)
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch history"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"messages":    messages,
		"next_cursor": nextCursor,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *FeedHandler) GetFeedStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	byteStream, qName, err := h.svc.GetStream()
	if err != nil {
		http.Error(w, "Failed to connect to stream", http.StatusInternalServerError)
		return
	}

	// Clean up the queue
	defer h.svc.CloseStream(qName)

	fmt.Fprintf(w, "event: connected\ndata: {\"status\": \"listening\"}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msgData := <-byteStream:
			// Write to the SSE stream
			fmt.Fprintf(w, "data: %s\n\n", string(msgData))
			flusher.Flush()
		}
	}
}
