package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/streadway/amqp"
)

type Order struct {
	ID    int    `json:"id"`
	Item  string `json:"item"`
	Price int    `json:"price"`
	User  string `json:"user"`
}

func main() {
	app := fiber.New()

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

	// Queue'leri tanımla
	orderQueue, _ := ch.QueueDeclare("order_created", true, false, false, false, nil)
	inventoryQueue, _ := ch.QueueDeclare("inventory_check", true, false, false, false, nil)
	paymentQueue, _ := ch.QueueDeclare("payment_process", true, false, false, false, nil)
	orderRollbackQueue, _ := ch.QueueDeclare("order_rollback", true, false, false, false, nil)

	app.Post("/order", func(c *fiber.Ctx) error {
		order := new(Order)
		if err := c.BodyParser(order); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
		}

		// Sipariş eventini RabbitMQ'ya gönder
		body, _ := json.Marshal(order)
		if err := ch.Publish("", orderQueue.Name, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Order process failed"})
		}

		if err := ch.Publish("", inventoryQueue.Name, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Inventory check failed"})
		}

		if err = ch.Publish("", paymentQueue.Name, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Payment process failed"})

		}

		if err := ch.Publish("", orderRollbackQueue.Name, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Order rollback failed"})
		}

		fmt.Printf("🚀 Sipariş alındı: Kullanıcı %s, Ürün: %s, Fiyat: %d TL\n", order.User, order.Item, order.Price)

		return c.JSON(order)
	})

	log.Fatal(app.Listen(":6000"))
}
