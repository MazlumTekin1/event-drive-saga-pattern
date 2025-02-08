package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

type Order struct {
	ID    int    `json:"id"`
	Item  string `json:"item"`
	Price int    `json:"price"`
}

func main() {
	// RabbitMQ'ya bağlan
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"order_created",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Ödeme işlemini simüle et
	go func() {
		for d := range msgs {
			var order Order
			json.Unmarshal(d.Body, &order)

			fmt.Printf("✅ Ödeme alındı: Sipariş #%d (%s) - %d TL\n", order.ID, order.Item, order.Price)
		}
	}()

	log.Println("Payment Service started, waiting for messages...")
	select {}
}
