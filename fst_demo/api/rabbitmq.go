package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	models "example.com/fst_demo/db"

	amqp "github.com/rabbitmq/amqp091-go"
)

const exchangeName = "listings"

// HouseEvent is the sanitized view of a House published to RabbitMQ.
// Description is intentionally omitted (may contain personal contact info).
type HouseEvent struct {
	ID          string          `json:"id"`
	PriceBucket string          `json:"price_bucket"`
	Location    models.Location `json:"location"`
	Topology    string          `json:"topology"`
}

func connectRabbitMQ(url string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("open channel: %w", err)
	}

	if err := declareExchange(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, err
	}

	log.Printf("rabbitmq: connected, exchange=%q declared", exchangeName)
	return conn, ch, nil
}

func declareExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		exchangeName,
		"headers",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	)
}

// publishHouseEvent publishes a sanitized house event to the headers exchange.
func publishHouseEvent(ch *amqp.Channel, house models.House) error {
	event := HouseEvent{
		ID:          house.ID,
		PriceBucket: house.PriceBucket,
		Location:    house.Location,
		Topology:    house.Topology,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal house event: %w", err)
	}

	headers := amqp.Table{
		"country":      strings.ToLower(house.Location.Country),
		"region":       strings.ToLower(house.Location.Region),
		"city":         strings.ToLower(house.Location.City),
		"topology":     strings.ToLower(house.Topology),
		"price_bucket": house.PriceBucket,
	}

	if err := ch.Publish(exchangeName, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("publish house event: %w", err)
	}

	log.Printf("rabbitmq: published house event id=%s price_bucket=%s city=%s",
		house.ID, house.PriceBucket, house.Location.City)
	return nil
}

// bindFilterQueue declares a queue for the filter and binds it to the headers
// exchange using x-match:all so every header must match exactly.
func bindFilterQueue(ch *amqp.Channel, filter models.Filter) error {
	_, err := ch.QueueDeclare(filter.QueueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue %q: %w", filter.QueueName, err)
	}

	buckets := models.BucketsInRange(filter.MinPrice, filter.MaxPrice)
	for _, bucket := range buckets {
		bindArgs := amqp.Table{
			"x-match":      "all",
			"country":      strings.ToLower(filter.Location.Country),
			"region":       strings.ToLower(filter.Location.Region),
			"city":         strings.ToLower(filter.Location.City),
			"topology":     strings.ToLower(filter.Topology),
			"price_bucket": bucket,
		}
		if err := ch.QueueBind(filter.QueueName, "", exchangeName, false, bindArgs); err != nil {
			return fmt.Errorf("bind queue %q for bucket %s: %w", filter.QueueName, bucket, err)
		}
	}

	log.Printf("rabbitmq: queue %q bound (%d bucket(s))", filter.QueueName, len(buckets))
	return nil
}
