package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
	"os"
	"time"
)

func main() {
	l, _ := zap.NewProduction()
	logger := l.Sugar()

	rabbitMqHost := os.Getenv("RABBITMQ_HOST")
	rabbitMqPort := os.Getenv("RABBITMQ_PORT")
	rabbitMqQueue := os.Getenv("RABBITMQ_QUEUE")

	// Init RabbitMQ client
	conn, err := amqp.Dial(fmt.Sprintf("amqp://guest:guest@%s:%s/", rabbitMqHost, rabbitMqPort))
	if err != nil {
		logger.Fatalf("Error connecting to RabbitMQ at %s on port %s", rabbitMqHost, rabbitMqPort)
	}
	ch, err := conn.Channel()
	if err != nil {
		logger.Fatal("Error creating RabbitMQ channel")
	}
	if err := ch.Qos(1, 0, true); err != nil { // don't dispatch a new message to a worker until it has processed and acknowledged the previous one
		logger.Fatalf("Error setting QOS on RabbitMQ channel")
	}
	defer conn.Close()
	defer ch.Close()

	// Init queue
	q, err := ch.QueueDeclare(
		rabbitMqQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Fatalf("Failed to declare queue %s", rabbitMqQueue)
	}

	// Publish messages
	for {
		body := fmt.Sprintf("Hello World! Time is %s", time.Now())
		err = ch.Publish(
			"",
			q.Name,
			false,
			false,
			amqp.Publishing{
				ContentType:  "text/plain",
				Body:         []byte(body),
				DeliveryMode: amqp.Persistent,
			},
		)
		if err != nil {
			logger.Error("Error publishing message")
		} else {
			logger.Infof("Message published to queue %s", rabbitMqQueue)
		}
		logger.Info("Now sleeping...")
		time.Sleep(1 * time.Second)
	}
}
