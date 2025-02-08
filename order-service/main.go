package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

type Order struct {
	ID    int    `json:"id"`
	Item  string `json:"item"`
	Price int    `json:"price"`
}

var db *sql.DB

func main() {
	// PostgreSQL bağlantısını aç
	var err error
	db, err = sql.Open("postgres", "postgres://user:password@localhost:5432/orders?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Fiber başlat
	app := fiber.New()

	// RabbitMQ bağlantısını aç
	rabbitConn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitConn.Close()

	ch, err := rabbitConn.Channel()
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

	// Yeni sipariş oluşturma endpointi
	app.Post("/order", func(c *fiber.Ctx) error {
		order := new(Order)
		if err := c.BodyParser(order); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
		}

		// Siparişi veritabanına kaydet
		err := db.QueryRow("INSERT INTO orders (item, price) VALUES ($1, $2) RETURNING id", order.Item, order.Price).Scan(&order.ID)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
		}

		// RabbitMQ'ya mesaj gönder
		body, _ := json.Marshal(order)
		err = ch.Publish(
			"",
			q.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "RabbitMQ error"})
		}

		return c.JSON(order)
	})

	log.Fatal(app.Listen(":5000"))
}
