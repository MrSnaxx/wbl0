package main

import (
    "sync"
    
)

type Cache struct {
    orders map[string]Order
    mu     sync.RWMutex
}

func NewCache() *Cache {
    return &Cache{
        orders: make(map[string]Order),
    }
}

// Загрузка данных в кэш (используется при старте)
func (c *Cache) Load(orders map[string]Order) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.orders = orders
}

// Получение заказа из кэша
func (c *Cache) GetOrder(orderUID string) (Order, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    order, exists := c.orders[orderUID]
    return order, exists
}

// Добавление заказа в кэш
func (c *Cache) SetOrder(order Order) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.orders[order.OrderUID] = order
}