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
	User  string `json:"user"`
}

var stock = map[string]int{
	"iPhone 15": 9, // Toplam 9 ürün var!
}

func main() {
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

	q, err := ch.QueueDeclare("inventory_check", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	paymentQueue, _ := ch.QueueDeclare("payment_process", true, false, false, false, nil)
	orderRollbackQueue, _ := ch.QueueDeclare("order_rollback", true, false, false, false, nil)

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for d := range msgs {
			var order Order
			json.Unmarshal(d.Body, &order)

			if stock[order.Item] > 0 {
				stock[order.Item]--

				// Ödeme servisine event gönder
				body, _ := json.Marshal(order)
				ch.Publish("", paymentQueue.Name, false, false, amqp.Publishing{
					ContentType: "application/json",
					Body:        body,
				})

				fmt.Printf("✅ Stok düşüldü: %s, Kalan: %d\n", order.Item, stock[order.Item])
			} else {
				// Stok yoksa rollback yap
				ch.Publish("", orderRollbackQueue.Name, false, false, amqp.Publishing{
					ContentType: "application/json",
					Body:        d.Body,
				})
				fmt.Printf("❌ Stok Yetersiz: %s için sipariş iptal edildi.\n", order.Item)
			}
		}
	}()

	log.Println("Inventory Service started, waiting for messages...")
	select {}
}
