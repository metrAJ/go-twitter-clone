package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQ(dsn string) (*RabbitMQ, error) {
	var conn *amqp.Connection
	var err error

	// The Retry Loop 5 attempts
	for i := 1; i <= 5; i++ {
		conn, err = amqp.Dial(dsn)
		if err == nil {
			break
		}
		log.Printf("RabbitMQ not ready yet (Attempt %d/5). Retrying in 5 seconds.", i)
		time.Sleep(5 * time.Second)
	}

	// If it still fails after 5 attempts, then it actually crash.
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ after retries: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		"twitter_feed", "fanout", true, false, false, false, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	log.Println(" Successfully connected to RabbitMQ and declared exchange!")
	return &RabbitMQ{
		conn:    conn,
		channel: ch,
	}, nil
}

func (r *RabbitMQ) Publish(ctx context.Context, body []byte) error {
	return r.channel.PublishWithContext(ctx,
		"twitter_feed",
		"",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (r *RabbitMQ) ConsumeWorker() (<-chan amqp.Delivery, error) {
	q, err := r.channel.QueueDeclare(
		"twitter_worker_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	err = r.channel.QueueBind(q.Name, "", "twitter_feed", false, nil)
	if err != nil {
		return nil, err
	}

	// QoS - Prefetch Count
	err = r.channel.Qos(
		10,
		0,
		false,
	)
	if err != nil {
		return nil, err
	}

	return r.channel.Consume(
		q.Name,
		"",
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

func (r *RabbitMQ) ConsumeStream() (<-chan amqp.Delivery, string, error) {
	q, err := r.channel.QueueDeclare(
		"",
		false, // durable
		true,  // auto-delete
		true,  // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return nil, "", err
	}

	err = r.channel.QueueBind(q.Name, "", "twitter_feed", false, nil)
	if err != nil {
		return nil, "", err
	}

	msgs, err := r.channel.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)

	return msgs, q.Name, err
}

func (r *RabbitMQ) DeleteStreamQueue(queueName string) {
	r.channel.QueueDelete(queueName, false, false, false)
}

func (r *RabbitMQ) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}
