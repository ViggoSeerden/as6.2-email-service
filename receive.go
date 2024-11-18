package main

import (
	"context"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// func fib(n int) int {
// 	if n == 0 {
// 		return 0
// 	} else if n == 1 {
// 		return 1
// 	} else {
// 		return fib(n-1) + fib(n-2)
// 	}
// }

func main() {
	rabbitmqURL := os.Getenv("RABBITMQ")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://localhost"
	}
	conn, err := amqp.Dial(rabbitmqURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"send-email", // name
		false,        // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		5,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

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

	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf("Received message: %s", string(d.Body))

			// Create a new context for each message
			publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			sendMail(string(d.Body))
			response := "Successfully sent email!"

			err := ch.PublishWithContext(publishCtx,
				"",        // exchange
				d.ReplyTo, // routing key
				false,     // mandatory
				false,     // immediate
				amqp.Publishing{
					ContentType:   "text/plain",
					CorrelationId: d.CorrelationId,
					Body:          []byte(response),
				})
			if err != nil {
				log.Printf("Failed to respond: %s", err)
			}

			if ackErr := d.Ack(false); ackErr != nil {
				log.Printf("Failed to acknowledge message: %s", ackErr)
			}
		}
	}()

	log.Printf("Now listening...")
	<-forever
}
