package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"errors"
)

func main() {
    dbHost := "localhost"
    dbPort := 5432
    dbUser := "postgres"
    dbPass := "admin"
    dbName := "wbl0"
    kafkaBrokers := "localhost:9092"
    httpPort := 8081
    flag.Parse()
    var ErrServerClosed = errors.New("http: Server closed")
    // Настройка логгера
    logger := log.New(os.Stdout, "ORDER-SERVICE: ", log.Ldate|log.Ltime|log.Lshortfile)
    
    // Подключение к БД
    connString := buildConnString(dbHost, dbPort, dbUser, dbPass, dbName)
    ctx := context.Background()
    
    pg, err := NewPostgres(ctx, connString)
    if err != nil {
        logger.Fatalf("Ошибка подключения к БД: %v", err)
    }
    defer pg.Close()
    
    repo := NewOrderRepository(pg)
    
    // Создание и заполнение кэша
    cache := NewCache()
    logger.Println("Загрузка кэша...")
    ordersMap, err := repo.GetLastThreeOrders(ctx)
    if err != nil {
        logger.Fatalf("Ошибка загрузки кэша: %v", err)
    }
    cache.Load(ordersMap)
    logger.Printf("%v заказов загружено в кэш", len(ordersMap))
    
    kafkaConsumer := NewConsumer(
        []string{kafkaBrokers},
        repo,
        cache,
        logger,
    )
    
    // Создание HTTP сервера
    server := NewServer(httpPort, cache, repo, logger)
    
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        kafkaConsumer.Start(ctx)
    }()
    
    // Запуск HTTP сервера
    go func() {
        if err := server.Start(); err != nil && err != ErrServerClosed {
            logger.Fatalf("Ошибка запуска HTTP сервера: %v", err)
        }
    }()
    
    logger.Println("Сервис запущен")
    
    <-stop
    logger.Println("Остановка...")
    
    if err := kafkaConsumer.Close(); err != nil {
        logger.Printf("Ошибка закрытия консьюмера: %v", err)
    }
    
    // Остановка HTTP сервера
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := server.Shutdown(shutdownCtx); err != nil {
        logger.Printf("Ошибка остановки HTTP сервера: %v", err)
    }
    
    logger.Println("Сервис остановлен")
}

func buildConnString(host string, port int, user, password, dbname string) string {
    return fmt.Sprintf(
        "host=%v port=%v user=%v password=%v dbname=%v sslmode=disable",
        host, port, user, password, dbname,
    )
}