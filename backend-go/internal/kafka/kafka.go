package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer() (*Producer, error) {
	kafkaEnabled := strings.ToLower(os.Getenv("KAFKA_ENABLED"))
	if kafkaEnabled != "true" {
		log.Println("⚠️  Kafka disabled or not configured")
		return &Producer{writer: nil}, nil
	}

	brokers := os.Getenv("KAFKA_BROKER")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers),
		Topic:    "game-events",
		Balancer: &kafka.LeastBytes{},
	}

	log.Println("✅ Kafka producer initialized")
	return &Producer{writer: writer}, nil
}

func (p *Producer) ProduceEvent(eventType string, data interface{}) error {
	if p.writer == nil {
		return nil
	}

	event := map[string]interface{}{
		"type": eventType,
		"data": data,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(eventType),
			Value: eventBytes,
		},
	)

	if err != nil {
		log.Printf("Error producing Kafka event: %v", err)
		return err
	}

	return nil
}

func (p *Producer) Close() error {
	if p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
