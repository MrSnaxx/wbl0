package kafka

import (
	"context"
	"encoding/json"
	"l0/internal/cache"
	"l0/internal/db"
	"l0/internal/model"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader       *kafka.Reader
	repo         db.OrderStore
	cachedOrders cache.CacheRepository
	logger       *log.Logger
	val          *validator.Validate
}

func NewConsumer(brokers []string, repo db.OrderStore, cachedOrders cache.CacheRepository, validator *validator.Validate, logger *log.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   brokers,
		Topic:     "orders",
		GroupID:   "order-service",
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
		Partition: 0,
	})

	return &Consumer{
		reader:       reader,
		repo:         repo,
		cachedOrders: cachedOrders,
		logger:       logger,
		val:          validator,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	c.logger.Println("Запуск консьюмера...")

	for {
		select {
		case <-ctx.Done():
			c.logger.Println("Остановка консьюмера...")
			return
		default:
			message, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() == nil {
					c.logger.Printf("Ошибка чтения сообщения: %v", err)
				}
				continue
			}

			c.logger.Printf("Получено сообщение: offset=%v, time=%v\n", message.Offset, message.Time)

			// Обработка сообщения
			c.processMessage(ctx, message)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) {
	var order model.Order

	// Десериализация JSON
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		c.logger.Printf("Ошибка десериализации сообщения: %v. Сообщение: %v", err, string(msg.Value))
		return
	}

	// Валидация данных
	if c.val != nil {
		if err := c.val.Struct(order); err != nil {
			validationErrors := err.(validator.ValidationErrors)
			for _, e := range validationErrors {
				c.logger.Printf("Ошибка валидации в поле '%s': %s (значение: %v)",
					e.Field(), e.Tag(), e.Value())
			}
			return
		}
	}

	// Сохранение в БД
	if err := c.repo.SaveOrder(ctx, order); err != nil {
		c.logger.Printf("Ошибка сохранения заказа %v в БД: %v", order.OrderUID, err)
		return
	}

	// Обновление кэша
	c.cachedOrders.SetOrder(order)

	// Подтверждение обработки сообщения
	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		c.logger.Printf("Ошибка подтверждения сообщения: %v", err)
	}

	c.logger.Printf("Заказ успешно обработан: %v", order.OrderUID)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
