package cache

import (
    "sync"
    "l0/internal/model"
)

type Cache struct {
    orders     map[string]model.Order
    orderList  []string // для отслеживания порядка добавления
    maxSize    int
    mu         sync.RWMutex
}

func NewCache() *Cache {
    return &Cache{
        orders:    make(map[string]model.Order),
        orderList: make([]string, 0),
        maxSize:   10,
    }
}

// Загрузка данных в кэш (используется при старте)
func (c *Cache) Load(orders map[string]model.Order) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.orders = orders
    // Пересоздаем список порядка добавления
    c.orderList = make([]string, 0, len(orders))
    for uid := range orders {
        c.orderList = append(c.orderList, uid)
    }
    c.evictIfNeeded()
}

// Получение заказа из кэша
func (c *Cache) GetOrder(orderUID string) (model.Order, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    order, exists := c.orders[orderUID]
    return order, exists
}

// Добавление заказа в кэш
func (c *Cache) SetOrder(order model.Order) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Если запись новая (не обновление существующей), добавляем в список
    if _, exists := c.orders[order.OrderUID]; !exists {
        c.orderList = append(c.orderList, order.OrderUID)
    }
    
    c.orders[order.OrderUID] = order
    c.evictIfNeeded()
}

// Удаление старых записей, если превышен лимит
func (c *Cache) evictIfNeeded() {
    for len(c.orders) > c.maxSize {
        // Удаляем самую старую запись
        oldestUID := c.orderList[0]
        delete(c.orders, oldestUID)
        
        // Удаляем из списка порядка
        c.orderList = c.orderList[1:]
    }
}