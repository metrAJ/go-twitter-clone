package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

func main() {
	// Configuration
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:3000/message"
	}

	rateStr := os.Getenv("MESSAGES_PER_SECOND")
	msgsPerSec, err := strconv.Atoi(rateStr)
	if err != nil || msgsPerSec <= 0 {
		msgsPerSec = 5 // Default to 5 messages a second
	}

	log.Printf("Starting Bot. Sending %d messages per second to %s", msgsPerSec, apiURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// The Ticker
	ticker := time.NewTicker(time.Second / time.Duration(msgsPerSec))
	defer ticker.Stop()

	counter := 1
BotLoop:
	for {
		select {
		case <-stop:
			log.Println("Stop signal received. Shutting down bot")
			cancel()
			break BotLoop

		case <-ticker.C:
			wg.Add(1)
			// Fire the request in a separate Goroutine so it doesnt block the next tick
			go func(msgID int) {
				defer wg.Done()

				payload := map[string]string{
					"content": fmt.Sprintf("Automated Tweet #%d generated at %v", msgID, time.Now().Format("15:04:05.000")),
				}

				body, _ := json.Marshal(payload)
				// Create the request
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(body))
				if err != nil {
					return
				}
				req.Header.Set("Content-Type", "application/json")
				// Execute the request
				resp, err := client.Do(req)
				if err != nil {
					if ctx.Err() == nil {
						log.Printf("Failed to send msg #%d: %v", msgID, err)
					}
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusAccepted {
					log.Printf("Sent msg #%d", msgID)
				} else {
					log.Printf("API returned status %d for msg #%d", resp.StatusCode, msgID)
				}
			}(counter)

			counter++
		}
	}

	wg.Wait()
	log.Println("Bot cleanly exited.")
}
