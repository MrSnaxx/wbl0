package cache

import (
	"l0/internal/model"
	"sync"
)

type CacheRepository interface {
	Load(orders map[string]model.Order)
	GetOrder(orderUID string) (model.Order, bool)
	SetOrder(order model.Order)
	evictIfNeeded()
}

type Cache struct {
	orders      *sync.Map // Используем указатель для безопасной замены
	orderList   []string
	orderListMu sync.Mutex
	maxSize     int
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		orders:    &sync.Map{},
		orderList: make([]string, 0),
		maxSize:   maxSize,
	}
}

func (c *Cache) Load(orders map[string]model.Order) {
	newOrders := &sync.Map{}
	for uid, order := range orders {
		newOrders.Store(uid, order)
	}
	c.orders = newOrders // Атомарная замена указателя

	c.orderListMu.Lock()
	defer c.orderListMu.Unlock()
	c.orderList = make([]string, 0, len(orders))
	for uid := range orders {
		c.orderList = append(c.orderList, uid)
	}
	c.evictIfNeeded()
}

func (c *Cache) GetOrder(orderUID string) (model.Order, bool) {
	value, ok := c.orders.Load(orderUID)
	if !ok {
		return model.Order{}, false
	}
	order, ok := value.(model.Order)
	return order, ok
}

func (c *Cache) SetOrder(order model.Order) {
	_, exists := c.orders.Load(order.OrderUID)
	if !exists {
		c.orderListMu.Lock()
		defer c.orderListMu.Unlock()
		c.orderList = append(c.orderList, order.OrderUID)
	}
	c.orders.Store(order.OrderUID, order)
	c.evictIfNeeded()
}

func (c *Cache) evictIfNeeded() {
	for len(c.orderList) > c.maxSize {
		oldestUID := c.orderList[0]
		c.orders.Delete(oldestUID)
		c.orderList = c.orderList[1:]
	}
}
