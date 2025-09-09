# Order Service (L0)

Сервис для обработки и хранения заказов, получаемых из Kafka, с предоставлением доступа через HTTP API.

## Функциональность

- **Потребление заказов** из Apache Kafka
- **Сохранение заказов** в PostgreSQL
- **In-memory кэширование** заказов для быстрого доступа
- **HTTP API** для получения информации о заказах
- **Автоматическое восстановление** кэша при запуске

## Архитектура
Kafka → Consumer → PostgreSQL → Cache ↔ HTTP Server

## Запуск сервиса

### Требования

- Go 1.18+
- PostgreSQL
- Apache Kafka

### Переменные окружения

Создайте файл `.env` или установите переменные окружения:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASS=your_password  # Обязательный параметр
DB_NAME=wbl0
KAFKA_BROKERS=localhost:9092
HTTP_PORT=8081
```

### Запуск
```bash
go run main.go
```

## API Endpoints

### Получить заказ по ID

```text
GET /orders/{id}
```

#### Ответ

```json
{
  "order_uid": "b563feb7b2b84b6test",
  "track_number": "WBILMTESTTRACK",
  "entry": "WBIL",
  "delivery": {
    "name": "Test Testov",
    "phone": "+9720000000",
    "zip": "2639809",
    "city": "Kiryat Mozkin",
    "address": "Ploshad Mira 15",
    "region": "Kraiot",
    "email": "test@gmail.com"
  },
  "payment": {
    "transaction": "b563feb7b2b84b6test",
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 9934930,
      "track_number": "WBILMTESTTRACK",
      "price": 453,
      "rid": "ab4219087a764ae0btest",
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389212,
      "brand": "Vivienne Sabo",
      "status": 202
    }
  ],
  "locale": "en",
  "internal_signature": "",
  "customer_id": "test",
  "delivery_service": "meest",
  "shardkey": "9",
  "sm_id": 99,
  "date_created": "2021-11-26T06:22:19Z",
  "oof_shard": "1"
}
```

## Конфигурация

### Обязательные параметры

- **DB_PASS** - пароль для подключения к PostgreSQL

### Опциональные параметры

- **DB_HOST** (localhost)
- **DB_PORT** (5432)
- **DB_USER** (postgres)
- **DB_NAME** (wbl0)
- **KAFKA_BROKERS** (localhost:9092)
- **HTTP_PORT** (8081)

## Особенности реализации

- Автоматическая загрузка последних 3 заказов в кэш при запуске
- Graceful shutdown при получении сигналов завершения
- Логирование ключевых событий работы сервиса
- Проверка обязательных параметров конфигурации