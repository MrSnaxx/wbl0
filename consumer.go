package main

import (
    "context"
    "encoding/json"
    "log"
    
    "github.com/segmentio/kafka-go"
)

type Consumer struct {
    reader   *kafka.Reader
    repo     *OrderRepository
    cache    *Cache
    logger   *log.Logger
}

func NewConsumer(brokers []string, repo *OrderRepository, cache *Cache, logger *log.Logger) *Consumer {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers: brokers,
        Topic: "orders",
        GroupID: "order-service",
        MinBytes: 10e3, // 10KB
        MaxBytes: 10e6, // 10MB
        Partition: 0,
    })
    
    return &Consumer{
        reader: reader,
        repo: repo,
        cache: cache,
        logger: logger,
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
                if ctx.Err() == nil { // Не ошибка контекста (т.е. не остановка)
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
    var order Order
    
    // Десериализация JSON
    if err := json.Unmarshal(msg.Value, &order); err != nil {
        c.logger.Printf("Error unmarshalling message: %v. Message: %v", err, string(msg.Value))
        // Не подтверждаем сообщение, чтобы оно вернулось в очередь
        return
    }
    
    // Валидация минимально необходимых данных
    if order.OrderUID == "" {
        c.logger.Printf("Invalid order: missing order_uid. Message: %v", string(msg.Value))
        return
    }
    
    // Сохранение в БД
    if err := c.repo.SaveOrder(ctx, order); err != nil {
        c.logger.Printf("Error saving order %v to database: %v", order.OrderUID, err)
        // Не подтверждаем сообщение, чтобы оно вернулось в очередь
        return
    }
    
    // Обновление кэша
    c.cache.SetOrder(order)
    
    // Подтверждение обработки сообщения
    if err := c.reader.CommitMessages(ctx, msg); err != nil {
        c.logger.Printf("Error committing message: %v", err)
    }
    
    c.logger.Printf("Заказ успешно обработан: %v", order.OrderUID)
}

func (c *Consumer) Close() error {
    return c.reader.Close()
}