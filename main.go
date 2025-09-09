package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"l0/internal/cache"
	"l0/internal/db"
	"l0/internal/http"
	"l0/internal/kafka"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func getEnv(key, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
}


// Получает ОБЯЗАТЕЛЬНУЮ переменную окружения
func getRequiredEnv(key string) string {
    value := os.Getenv(key)
    if value == "" {
        log.Fatalf("Ошибка: обязательная переменная %v не установлена", key)
    }
    return value
}

// Получает переменную окружения как целое число
func getEnvAsInt(key string, defaultValue int) int {
    strValue := getEnv(key, "")
    if strValue == "" {
        return defaultValue
    }
    
    value, err := strconv.Atoi(strValue)
    if err != nil {
        log.Fatalf("Неверный формат %v: %v", key, strValue)
    }
    return value
}

func main() {    
    // Загружаем .env только в локальной среде
    if err := godotenv.Load(); err != nil {
        log.Println("Локальный .env файл не найден, используем системные переменные")
    }

    // Получаем параметры из окружения с проверкой
    dbHost := getEnv("DB_HOST", "localhost")
    dbPort := getEnvAsInt("DB_PORT", 5432)
    dbUser := getEnv("DB_USER", "postgres")
    dbPass := getRequiredEnv("DB_PASS") // Обязательный параметр
    dbName := getEnv("DB_NAME", "wbl0")
    kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
    httpPort := getEnvAsInt("HTTP_PORT", 8081)

    flag.Parse()
    var ErrServerClosed = errors.New("http: Server closed")
    // Настройка логгера
    logger := log.New(os.Stdout, "ORDER-SERVICE: ", log.Ldate|log.Ltime|log.Lshortfile)
    
    // Подключение к БД
    connString := buildConnString(dbHost, dbPort, dbUser, dbPass, dbName)
    ctx := context.Background()
    
    pg, err := db.NewPostgres(ctx, connString)
    if err != nil {
        logger.Fatalf("Ошибка подключения к БД: %v", err)
    }
    defer pg.Close()
    
    repo := db.NewOrderRepository(pg)
    
    // Создание и заполнение кэша
    cache := cache.NewCache()
    logger.Println("Загрузка кэша...")
    ordersMap, err := repo.GetLastThreeOrders(ctx)
    if err != nil {
        logger.Fatalf("Ошибка загрузки кэша: %v", err)
    }
    cache.Load(ordersMap)
    logger.Printf("%v заказов загружено в кэш", len(ordersMap))
    
    kafkaConsumer := kafka.NewConsumer(
        []string{kafkaBrokers},
        repo,
        cache,
        logger,
    )
    
    // Создание HTTP сервера
    server := http.NewServer(httpPort, cache, repo, logger)
    
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