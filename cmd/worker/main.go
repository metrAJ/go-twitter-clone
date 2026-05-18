package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"twitter/internal/queue"
	"twitter/internal/repository"
	database "twitter/internal/storage"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Println("DATABASE_URL not set, defaulting to localhost CockroachDB...")
		dbURL = "postgres://root@localhost:26257/twitter?sslmode=disable"
	}

	pool, err := database.InitDB(ctx, dbURL)
	if err != nil {
		log.Fatalf("Worker DB Connection failed: %v", err)
	}
	defer pool.Close()
	repo := repository.NewCockroachRepo(pool)

	rmqURL := os.Getenv("RABBITMQ_URL")
	if rmqURL == "" {
		log.Println("RABBITMQ_URL not set, defaulting to localhost RabbitMQ...")
		rmqURL = "amqp://guest:guest@localhost:5672/"
	}

	mq, err := queue.NewRabbitMQ(rmqURL)
	if err != nil {
		log.Fatalf("Worker RabbitMQ Connection failed: %v", err)
	}
	defer mq.Close()

	msgs, err := mq.ConsumeWorker()
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Println("Worker is running and waiting for messages...")

	// The Worker Loop
	for {
		select {
		case msg := <-msgs:
			var incoming struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(msg.Body, &incoming); err != nil {
				log.Printf("Invalid message format: %v", err)
				msg.Nack(false, false)
				continue
			}

			_, err := repo.InsertMessage(ctx, incoming.Content)
			if err != nil {
				log.Printf("DB Insert failed: %v", err)
				msg.Nack(false, true)
				continue
			}

			msg.Ack(false)
			log.Println("Worker saved new message to DB")

		case <-stop:
			log.Println("Shutting down worker...")
			return
		}
	}
}
