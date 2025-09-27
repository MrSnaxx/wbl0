package db

import (
	"context"
	"l0/internal/model"
	"testing"
	"time"
)


type MockOrderStore struct {
	saveOrderFn      func(ctx context.Context, ord model.Order) error
	getOrderByIDFn   func(ctx context.Context, orderUID string) (*model.Order, error)
	getAllOrdersFn   func(ctx context.Context) (map[string]model.Order, error)
	getLastThreeFn   func(ctx context.Context) (map[string]model.Order, error)
}

func (m *MockOrderStore) SaveOrder(ctx context.Context, ord model.Order) error {
	if m.saveOrderFn != nil {
		return m.saveOrderFn(ctx, ord)
	}
	return nil
}

func (m *MockOrderStore) GetOrderByID(ctx context.Context, orderUID string) (*model.Order, error) {
	if m.getOrderByIDFn != nil {
		return m.getOrderByIDFn(ctx, orderUID)
	}
	return nil, nil
}

func (m *MockOrderStore) GetAllOrders(ctx context.Context) (map[string]model.Order, error) {
	if m.getAllOrdersFn != nil {
		return m.getAllOrdersFn(ctx)
	}
	return map[string]model.Order{}, nil
}

func (m *MockOrderStore) GetLastThreeOrders(ctx context.Context) (map[string]model.Order, error) {
	if m.getLastThreeFn != nil {
		return m.getLastThreeFn(ctx)
	}
	return map[string]model.Order{}, nil
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
			Transaction:  "txn-" + uid,
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
				ChrtID:      123456 + len(uid),
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


func TestMockOrderStore_SaveOrder(t *testing.T) {
	called := false
	mock := &MockOrderStore{
		saveOrderFn: func(ctx context.Context, ord model.Order) error {
			called = true
			if ord.OrderUID != "test-123" {
				t.Errorf("Expected OrderUID 'test-123', got %s", ord.OrderUID)
			}
			return nil
		},
	}

	err := mock.SaveOrder(context.Background(), newValidOrder("test-123"))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !called {
		t.Error("SaveOrder was not called")
	}
}

func TestMockOrderStore_GetOrderByID(t *testing.T) {
	expected := newValidOrder("test-456")
	mock := &MockOrderStore{
		getOrderByIDFn: func(ctx context.Context, uid string) (*model.Order, error) {
			if uid == "test-456" {
				return &expected, nil
			}
			return nil, nil
		},
	}

	order, err := mock.GetOrderByID(context.Background(), "test-456")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if order == nil || order.OrderUID != "test-456" {
		t.Errorf("Expected order 'test-456', got %+v", order)
	}
}

func TestMockOrderStore_GetAllOrders(t *testing.T) {
	expected := map[string]model.Order{
		"1": newValidOrder("1"),
		"2": newValidOrder("2"),
	}
	mock := &MockOrderStore{
		getAllOrdersFn: func(ctx context.Context) (map[string]model.Order, error) {
			return expected, nil
		},
	}

	orders, err := mock.GetAllOrders(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("Expected 2 orders, got %d", len(orders))
	}
	if orders["1"].OrderUID != "1" {
		t.Error("Order '1' mismatch")
	}
}

type OrderService struct {
	store OrderStore
}

func NewOrderService(store OrderStore) *OrderService {
	return &OrderService{store: store}
}

func (s *OrderService) ProcessOrder(ctx context.Context, uid string) error {
	_, err := s.store.GetOrderByID(ctx, uid)
	return err
}

func TestOrderService_WithMock(t *testing.T) {
	mock := &MockOrderStore{
		getOrderByIDFn: func(ctx context.Context, uid string) (*model.Order, error) {
			return &model.Order{OrderUID: uid}, nil
		},
	}

	service := NewOrderService(mock)
	err := service.ProcessOrder(context.Background(), "mocked-uid")
	if err != nil {
		t.Errorf("ProcessOrder failed: %v", err)
	}
}
