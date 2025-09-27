package cache

import (
	"l0/internal/model"
	"sync"
	"testing"
	"time"
)

type MockCacheRepository struct {
	loadFn      func(orders map[string]model.Order)
	getOrderFn  func(orderUID string) (model.Order, bool)
	setOrderFn  func(order model.Order)
	evictFn     func()
}

func (m *MockCacheRepository) Load(orders map[string]model.Order) {
	if m.loadFn != nil {
		m.loadFn(orders)
	}
}

func (m *MockCacheRepository) GetOrder(orderUID string) (model.Order, bool) {
	if m.getOrderFn != nil {
		return m.getOrderFn(orderUID)
	}
	return model.Order{}, false
}

func (m *MockCacheRepository) SetOrder(order model.Order) {
	if m.setOrderFn != nil {
		m.setOrderFn(order)
	}
}

func (m *MockCacheRepository) evictIfNeeded() {
	if m.evictFn != nil {
		m.evictFn()
	}
}

func newValidOrder(uid string) model.Order {
	return model.Order{
		OrderUID:          uid,
		TrackNumber:       "WB0000000001",
		Entry:             "WB",
		Locale:            "ru",
		InternalSignature: "sig123",
		CustomerID:        "customer-123",
		DeliveryService:   "WB",
		Shardkey:          "123",
		SMID:              99,
		DateCreated:       time.Now(),
		OofShard:          "1",
		Delivery: model.Delivery{
			Name:    "Иван Иванов",
			Phone:   "89001234567",
			Zip:     "123456",
			City:    "Москва",
			Address: "ул. Ленина, д. 1, кв. 1",
			Region:  "Московская область",
			Email:   "ivan@example.com",
		},
		Payment: model.Payment{
			Transaction:  "1234567890",
			RequestID:    "req123",
			Currency:     "RUB",
			Provider:     "wb",
			Amount:       1000.0,
			PaymentDT:    time.Now().Unix(),
			Bank:         "wb-bank",
			DeliveryCost: 100.0,
			GoodsTotal:   900,
			CustomFee:    0.0,
		},
		Items: []model.Item{
			{
				ChrtID:      123456,
				TrackNumber: "WB0000000001",
				Price:       900.0,
				RID:         "rid123",
				Name:        "Тестовый товар",
				Sale:        10.0,
				Size:        "M",
				TotalPrice:  810.0,
				NMID:        654321,
				Brand:       "BrandX",
				Status:      200,
			},
		},
	}
}

// ТЕСТЫ

func TestCache_Load(t *testing.T) {
	cache := NewCache(10)
	orders := map[string]model.Order{
		"1": newValidOrder("1"),
		"2": newValidOrder("2"),
	}

	cache.Load(orders)

	if order, ok := cache.GetOrder("1"); !ok || order.OrderUID != "1" {
		t.Errorf("Expected order '1', got %+v", order)
	}
	if order, ok := cache.GetOrder("2"); !ok || order.OrderUID != "2" {
		t.Errorf("Expected order '2', got %+v", order)
	}
	if _, ok := cache.GetOrder("3"); ok {
		t.Error("Expected no order with UID '3'")
	}
}

func TestCache_SetOrder(t *testing.T) {
	cache := NewCache(10)
	order := newValidOrder("100")

	cache.SetOrder(order)

	got, ok := cache.GetOrder("100")
	if !ok {
		t.Fatal("Order not found after SetOrder")
	}
	if got.OrderUID != "100" {
		t.Errorf("Expected OrderUID '100', got %s", got.OrderUID)
	}
}

func TestCache_Eviction(t *testing.T) {
	maxSize := 2
	cache := NewCache(maxSize)

	cache.SetOrder(newValidOrder("1"))
	cache.SetOrder(newValidOrder("2"))
	cache.SetOrder(newValidOrder("3"))

	if _, ok := cache.GetOrder("1"); ok {
		t.Error("Order '1' should have been evicted")
	}
	if _, ok := cache.GetOrder("2"); !ok {
		t.Error("Order '2' should be present")
	}
	if _, ok := cache.GetOrder("3"); !ok {
		t.Error("Order '3' should be present")
	}

	cache.orderListMu.Lock()
	defer cache.orderListMu.Unlock()
	if len(cache.orderList) != maxSize {
		t.Errorf("Expected orderList length %d, got %d", maxSize, len(cache.orderList))
	}
	if cache.orderList[0] != "2" || cache.orderList[1] != "3" {
		t.Errorf("Expected orderList [2,3], got %v", cache.orderList)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache(100)
	var wg sync.WaitGroup
	const numGoroutines = 10
	const ordersPerGoroutine = 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < ordersPerGoroutine; j++ {
				uid := string(rune('A'+goroutineID)) + string(rune('0'+j))
				cache.SetOrder(newValidOrder(uid))
				_, _ = cache.GetOrder(uid)
			}
		}(i)
	}

	wg.Wait()

	cache.orderListMu.Lock()
	count := len(cache.orderList)
	cache.orderListMu.Unlock()

	if count > numGoroutines*ordersPerGoroutine {
		t.Errorf("Too many orders stored: %d", count)
	}
}

func TestCache_GetOrder_NotFound(t *testing.T) {
	cache := NewCache(5)
	_, ok := cache.GetOrder("nonexistent")
	if ok {
		t.Error("Expected order not to be found")
	}
}

func TestUsingMockCacheRepository(t *testing.T) {
	mock := &MockCacheRepository{
		getOrderFn: func(uid string) (model.Order, bool) {
			if uid == "test-uid" {
				return newValidOrder("test-uid"), true
			}
			return model.Order{}, false
		},
	}

	order, ok := mock.GetOrder("test-uid")
	if !ok || order.OrderUID != "test-uid" {
		t.Errorf("Mock returned unexpected order: %+v", order)
	}
}