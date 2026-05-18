package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"twitter/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageStore interface {
	InsertMessage(ctx context.Context, content string) (*models.Message, error)
	GetRecentMessages(ctx context.Context, limit int) ([]*models.Message, error)
	GetFeedCursor(ctx context.Context, limit int, cursorTime time.Time, cursorID string) ([]*models.Message, error)
}

type EventBroker interface {
	Publish(ctx context.Context, body []byte) error
	ConsumeStream() (<-chan amqp.Delivery, string, error)
	DeleteStreamQueue(queueName string)
}

type Cursor interface {
	Encode(t time.Time, id string) string
	Decode(encoded string) (time.Time, string, error)
}

type Twitter struct {
	store  MessageStore
	broker EventBroker
	cursor Cursor
}

func NewService(store MessageStore, broker EventBroker, cursor Cursor) *Twitter {
	return &Twitter{
		store:  store,
		broker: broker,
		cursor: cursor,
	}
}

func (s *Twitter) CreateTweet(ctx context.Context, content string) error {
	payload := map[string]string{
		"content":    content,
		"created_at": time.Now().Format(time.RFC3339),
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal tweet payload: %w", err)
	}

	return s.broker.Publish(ctx, bodyBytes)
}

func (s *Twitter) GetTweets(ctx context.Context, limit int, cursorStr string) ([]*models.Message, string, error) {
	var messages []*models.Message
	var err error

	if cursorStr == "" {
		messages, err = s.store.GetRecentMessages(ctx, limit)
	} else {
		cTime, cID, decodeErr := s.cursor.Decode(cursorStr)
		if decodeErr != nil {
			return nil, "", fmt.Errorf("invalid cursor format")
		}
		messages, err = s.store.GetFeedCursor(ctx, limit, cTime, cID)
	}

	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		nextCursor = s.cursor.Encode(lastMsg.CreatedAt, lastMsg.ID)
	}

	return messages, nextCursor, nil
}

func (s *Twitter) GetStream() (<-chan []byte, string, error) {
	msgs, qName, err := s.broker.ConsumeStream()
	if err != nil {
		return nil, "", err
	}

	byteStream := make(chan []byte)
	go func() {
		for msg := range msgs {
			byteStream <- msg.Body
		}
	}()

	return byteStream, qName, nil
}

func (s *Twitter) CloseStream(queueName string) {
	s.broker.DeleteStreamQueue(queueName)
}
