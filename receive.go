package main

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	rabbitmqURL := os.Getenv("RABBITMQ")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://localhost"
	}

	// Connect to RabbitMQ
	conn, err := amqp.Dial(rabbitmqURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// Declare a topic exchange
	err = ch.ExchangeDeclare(
		"osso-exchange", // exchange name
		"topic",         // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	// Declare a queue
	q, err := ch.QueueDeclare(
		"",    // random queue name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// Bind the queue to multiple routing patterns
	err = ch.QueueBind(
		q.Name,              // queue name
		"email.request.*.*", // routing key
		"osso-exchange",     // exchange
		false,               // no-wait
		nil,                 // arguments
	)
	failOnError(err, "Failed to bind queue")

	// Set QoS
	err = ch.Qos(
		5,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	// Start consuming messages
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	// Process messages in a goroutine
	go func() {
		for d := range msgs {
			log.Printf("Received message with routing key '%s': %s", d.RoutingKey, string(d.Body))

			// Create a new context for each message
			publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			routingKeyParts := strings.Split(d.RoutingKey, ".")
			routingKeyBase := routingKeyParts[0] + "." + routingKeyParts[1]

			response := "Failed to send email to " + routingKeyParts[3]

			// Handle message based on routing key
			switch routingKeyBase {
			case "email.request":
				decodedEmail, err := base64.StdEncoding.DecodeString(routingKeyParts[3])
				if err != nil {
					log.Fatal("error:", err)
				}
				sendMail(string(decodedEmail), string(d.Body))
				response = "Successfully sent email to " + routingKeyParts[3]
			default:
				log.Printf("Unhandled routing key: %s", d.RoutingKey)
			}

			// Respond to the sender
			err := ch.PublishWithContext(publishCtx,
				"osso-exchange",                      // exchange
				"email.response."+routingKeyParts[2], // routing key
				false,                                // mandatory
				false,                                // immediate
				amqp.Publishing{
					ContentType:   "text/plain",
					CorrelationId: d.CorrelationId,
					Body:          []byte(response),
				})
			if err != nil {
				log.Printf("Failed to respond: %s", err)
			} else {
				log.Printf("Returned response '" + response + "' with routing key " + "email.response." + routingKeyParts[2])
			}

			// Acknowledge the message
			if ackErr := d.Ack(false); ackErr != nil {
				log.Printf("Failed to acknowledge message: %s", ackErr)
			}
		}
	}()

	log.Printf("Now listening...")
	// Keep the program running
	select {}
}
