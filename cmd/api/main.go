package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"twitter/internal/cursor"
	"twitter/internal/queue"
	"twitter/internal/repository"
	"twitter/internal/service"
	twitter_handler "twitter/internal/service/transport"
	database "twitter/internal/storage"
)

func main() {
	ctx := context.Background()

	log.Println("Booting up infrastructure...")

	// connect to DB
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://root@localhost:26257/twitter?sslmode=disable" // for local dev
	}
	pool, err := database.InitDB(ctx, dbURL)
	if err != nil {
		log.Fatalf("Fatal: Database connection failed: %v", err)
	}
	defer pool.Close()
	// Init Repo
	repo := repository.NewCockroachRepo(pool)
	// Connect to RabbitMQ
	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		rmqURL = "amqp://guest:guest@localhost:5672/" // for local dev
	}
	mq, err := queue.NewRabbitMQ(rmqURL)
	if err != nil {
		log.Fatalf("Fatal: RabbitMQ connection failed: %v", err)
	}
	defer mq.Close()

	// Init the Cursor
	codec := cursor.NewBase64Codec()
	// Init service
	twservice := service.NewService(repo, mq, codec)
	// Init HTTP handler
	httpHandler := twitter_handler.NewFeedHandler(twservice)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /message", httpHandler.PostMessage)
	mux.HandleFunc("GET /feed", httpHandler.GetRecentMessages)
	mux.HandleFunc("GET /feed/stream", httpHandler.GetFeedStream)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // for local dev
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start the server
	go func() {
		log.Printf("Server listening on http://localhost:%s", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Fatal: Server crashed: %v", err)
		}
	}()

	// Channel for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// The main function blocks here until a signal is received
	<-quit
	log.Println("Shutting down server")

	// Give 10 seconds to finish before killing
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly")
}
