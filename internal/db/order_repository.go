package db

import (
	"context"
	"fmt"
	"l0/internal/model"
)

type OrderStore interface {
	SaveOrder(ctx context.Context, ord model.Order) error
	GetOrderByID(ctx context.Context, orderUID string) (*model.Order, error)
	GetAllOrders(ctx context.Context) (map[string]model.Order, error)
}

type OrderRepository struct {
	db *Postgres
}

func NewOrderRepository(db *Postgres) *OrderRepository {
	return &OrderRepository{db: db}
}

// Сохранение заказа со всеми внутренностями в транзакции
func (r *OrderRepository) SaveOrder(ctx context.Context, ord model.Order) error {
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Заказ
	_, err = tx.Exec(ctx, `
        INSERT INTO orders (
            order_uid, track_number, entry, locale, internal_signature, 
            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (order_uid) DO UPDATE SET
            track_number = EXCLUDED.track_number,
            entry = EXCLUDED.entry,
            locale = EXCLUDED.locale,
            internal_signature = EXCLUDED.internal_signature,
            customer_id = EXCLUDED.customer_id,
            delivery_service = EXCLUDED.delivery_service,
            shardkey = EXCLUDED.shardkey,
            sm_id = EXCLUDED.sm_id,
            date_created = EXCLUDED.date_created,
            oof_shard = EXCLUDED.oof_shard
    `,
		ord.OrderUID, ord.TrackNumber, ord.Entry, ord.Locale, ord.InternalSignature,
		ord.CustomerID, ord.DeliveryService, ord.Shardkey, ord.SMID, ord.DateCreated, ord.OofShard)
	if err != nil {
		return fmt.Errorf("failed to save order: %v", err)
	}

	// Доставка
	_, err = tx.Exec(ctx, `
        INSERT INTO delivery (
            order_uid, name, phone, zip, city, address, region, email
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (order_uid) DO UPDATE SET
            name = EXCLUDED.name,
            phone = EXCLUDED.phone,
            zip = EXCLUDED.zip,
            city = EXCLUDED.city,
            address = EXCLUDED.address,
            region = EXCLUDED.region,
            email = EXCLUDED.email
    `,
		ord.OrderUID, ord.Delivery.Name, ord.Delivery.Phone, ord.Delivery.Zip,
		ord.Delivery.City, ord.Delivery.Address, ord.Delivery.Region, ord.Delivery.Email)
	if err != nil {
		return fmt.Errorf("failed to save delivery: %v", err)
	}

	// Оплата
	_, err = tx.Exec(ctx, `
        INSERT INTO payments (
            transaction, order_uid, request_id, currency, provider, amount, 
            payment_dt, bank, delivery_cost, goods_total, custom_fee
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (transaction) DO UPDATE SET
            order_uid = EXCLUDED.order_uid,
            request_id = EXCLUDED.request_id,
            currency = EXCLUDED.currency,
            provider = EXCLUDED.provider,
            amount = EXCLUDED.amount,
            payment_dt = EXCLUDED.payment_dt,
            bank = EXCLUDED.bank,
            delivery_cost = EXCLUDED.delivery_cost,
            goods_total = EXCLUDED.goods_total,
            custom_fee = EXCLUDED.custom_fee
    `,
		ord.Payment.Transaction, ord.OrderUID, ord.Payment.RequestID, ord.Payment.Currency,
		ord.Payment.Provider, ord.Payment.Amount, ord.Payment.PaymentDT,
		ord.Payment.Bank, ord.Payment.DeliveryCost, ord.Payment.GoodsTotal, ord.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("failed to save payment: %v", err)
	}

	// Товары
	for _, item := range ord.Items {
		// Товар
		_, err = tx.Exec(ctx, `
            INSERT INTO items (
                chrt_id, track_number, price, rid, name, sale, size, 
                total_price, nm_id, brand, status
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            ON CONFLICT (chrt_id) DO UPDATE SET
                track_number = EXCLUDED.track_number,
                price = EXCLUDED.price,
                rid = EXCLUDED.rid,
                name = EXCLUDED.name,
              	sale = EXCLUDED.sale,
                size = EXCLUDED.size,
                total_price = EXCLUDED.total_price,
                nm_id = EXCLUDED.nm_id,
                brand = EXCLUDED.brand,
                status = EXCLUDED.status
        `,
			item.ChrtID, item.TrackNumber, item.Price, item.RID, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NMID, item.Brand, item.Status)
		if err != nil {
			return fmt.Errorf("failed to save item: %v", err)
		}

		// Связь с заказом
		_, err = tx.Exec(ctx, `
            INSERT INTO order_items (order_uid, chrt_id)
            VALUES ($1, $2)
            ON CONFLICT (order_uid, chrt_id) DO NOTHING
        `, ord.OrderUID, item.ChrtID)
		if err != nil {
			return fmt.Errorf("failed to save order-item relation: %v", err)
		}
	}

	// Коммит транзакции
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// Получение заказа по ID
func (r *OrderRepository) GetOrderByID(ctx context.Context, orderUID string) (*model.Order, error) {
	// Получаем заказ
	var ord model.Order
	err := r.db.pool.QueryRow(ctx, `
        SELECT order_uid, track_number, entry, locale, internal_signature, 
               customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders
        WHERE order_uid = $1
    `, orderUID).Scan(
		&ord.OrderUID, &ord.TrackNumber, &ord.Entry, &ord.Locale, &ord.InternalSignature,
		&ord.CustomerID, &ord.DeliveryService, &ord.Shardkey, &ord.SMID, &ord.DateCreated, &ord.OofShard)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %v", err)
	}

	// Получаем доставку
	err = r.db.pool.QueryRow(ctx, `
        SELECT name, phone, zip, city, address, region, email
        FROM delivery
        WHERE order_uid = $1
    `, orderUID).Scan(
		&ord.Delivery.Name, &ord.Delivery.Phone, &ord.Delivery.Zip,
		&ord.Delivery.City, &ord.Delivery.Address, &ord.Delivery.Region, &ord.Delivery.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery: %v", err)
	}

	// Получаем оплату
	err = r.db.pool.QueryRow(ctx, `
        SELECT transaction, request_id, currency, provider, amount, 
               payment_dt, bank, delivery_cost, goods_total, custom_fee
        FROM payments
        WHERE order_uid = $1
    `, orderUID).Scan(
		&ord.Payment.Transaction, &ord.Payment.RequestID, &ord.Payment.Currency,
		&ord.Payment.Provider, &ord.Payment.Amount, &ord.Payment.PaymentDT,
		&ord.Payment.Bank, &ord.Payment.DeliveryCost, &ord.Payment.GoodsTotal, &ord.Payment.CustomFee)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %v", err)
	}

	// Получаем товары
	rows, err := r.db.pool.Query(ctx, `
        SELECT i.chrt_id, i.track_number, i.price, i.rid, i.name, i.sale, i.size, 
               i.total_price, i.nm_id, i.brand, i.status
        FROM items i
        JOIN order_items oi ON i.chrt_id = oi.chrt_id
        WHERE oi.order_uid = $1
    `, orderUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.Item
		if err := rows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NMID, &item.Brand, &item.Status,
		); err != nil {
			return nil, fmt.Errorf("failed to scan item: %v", err)
		}
		ord.Items = append(ord.Items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating items: %v", err)
	}

	return &ord, nil
}

// Получение всех заказов (для заполнения кэша при старте)
func (r *OrderRepository) GetAllOrders(ctx context.Context) (map[string]model.Order, error) {
	ordersMap := make(map[string]model.Order)

	// Получаем все заказы
	rows, err := r.db.pool.Query(ctx, `
        SELECT order_uid, track_number, entry, locale, internal_signature, 
               customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to get all orders: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ord model.Order
		if err := rows.Scan(
			&ord.OrderUID, &ord.TrackNumber, &ord.Entry, &ord.Locale, &ord.InternalSignature,
			&ord.CustomerID, &ord.DeliveryService, &ord.Shardkey, &ord.SMID, &ord.DateCreated, &ord.OofShard,
		); err != nil {
			return nil, fmt.Errorf("failed to scan order: %v", err)
		}

		// Получаем доставку для каждого заказа
		err = r.db.pool.QueryRow(ctx, `
            SELECT name, phone, zip, city, address, region, email
            FROM delivery
            WHERE order_uid = $1
        `, ord.OrderUID).Scan(
			&ord.Delivery.Name, &ord.Delivery.Phone, &ord.Delivery.Zip,
			&ord.Delivery.City, &ord.Delivery.Address, &ord.Delivery.Region, &ord.Delivery.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to get delivery for order %v: %v", ord.OrderUID, err)
		}

		// Получаем оплату для каждого заказа
		err = r.db.pool.QueryRow(ctx, `
            SELECT transaction, request_id, currency, provider, amount, 
                   payment_dt, bank, delivery_cost, goods_total, custom_fee
            FROM payments
            WHERE order_uid = $1
        `, ord.OrderUID).Scan(
			&ord.Payment.Transaction, &ord.Payment.RequestID, &ord.Payment.Currency,
			&ord.Payment.Provider, &ord.Payment.Amount, &ord.Payment.PaymentDT,
			&ord.Payment.Bank, &ord.Payment.DeliveryCost, &ord.Payment.GoodsTotal, &ord.Payment.CustomFee)
		if err != nil {
			return nil, fmt.Errorf("failed to get payment for order %v: %v", ord.OrderUID, err)
		}

		// Получаем товары для каждого заказа
		itemRows, err := r.db.pool.Query(ctx, `
            SELECT i.chrt_id, i.track_number, i.price, i.rid, i.name, i.sale, i.size, 
                   i.total_price, i.nm_id, i.brand, i.status
            FROM items i
            JOIN order_items oi ON i.chrt_id = oi.chrt_id
            WHERE oi.order_uid = $1
        `, ord.OrderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get items for order %v: %v", ord.OrderUID, err)
		}

		for itemRows.Next() {
			var item model.Item
			if err := itemRows.Scan(
				&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name,
				&item.Sale, &item.Size, &item.TotalPrice, &item.NMID, &item.Brand, &item.Status,
			); err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("failed to scan item for order %v: %v", ord.OrderUID, err)
			}
			ord.Items = append(ord.Items, item)
		}
		itemRows.Close()

		ordersMap[ord.OrderUID] = ord
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %v", err)
	}

	return ordersMap, nil
}

func (r *OrderRepository) GetLastThreeOrders(ctx context.Context) (map[string]model.Order, error) { // Получаем три последних заказа по дате
	ordersMap := make(map[string]model.Order)

	rows, err := r.db.pool.Query(ctx, `
        SELECT order_uid, track_number, entry, locale, internal_signature, 
               customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders
        ORDER BY date_created DESC
        LIMIT 3
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to get last three orders: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ord model.Order
		if err := rows.Scan(
			&ord.OrderUID, &ord.TrackNumber, &ord.Entry, &ord.Locale, &ord.InternalSignature,
			&ord.CustomerID, &ord.DeliveryService, &ord.Shardkey, &ord.SMID, &ord.DateCreated, &ord.OofShard,
		); err != nil {
			return nil, fmt.Errorf("failed to scan order: %v", err)
		}

		err = r.db.pool.QueryRow(ctx, `
            SELECT name, phone, zip, city, address, region, email
            FROM delivery
            WHERE order_uid = $1
        `, ord.OrderUID).Scan(
			&ord.Delivery.Name, &ord.Delivery.Phone, &ord.Delivery.Zip,
			&ord.Delivery.City, &ord.Delivery.Address, &ord.Delivery.Region, &ord.Delivery.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to get delivery for order %v: %v", ord.OrderUID, err)
		}

		err = r.db.pool.QueryRow(ctx, `
            SELECT transaction, request_id, currency, provider, amount, 
                   payment_dt, bank, delivery_cost, goods_total, custom_fee
            FROM payments
            WHERE order_uid = $1
        `, ord.OrderUID).Scan(
			&ord.Payment.Transaction, &ord.Payment.RequestID, &ord.Payment.Currency,
			&ord.Payment.Provider, &ord.Payment.Amount, &ord.Payment.PaymentDT,
			&ord.Payment.Bank, &ord.Payment.DeliveryCost, &ord.Payment.GoodsTotal, &ord.Payment.CustomFee)
		if err != nil {
			return nil, fmt.Errorf("failed to get payment for order %v: %v", ord.OrderUID, err)
		}

		itemRows, err := r.db.pool.Query(ctx, `
            SELECT i.chrt_id, i.track_number, i.price, i.rid, i.name, i.sale, i.size, 
                   i.total_price, i.nm_id, i.brand, i.status
            FROM items i
            JOIN order_items oi ON i.chrt_id = oi.chrt_id
            WHERE oi.order_uid = $1
        `, ord.OrderUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get items for order %v: %v", ord.OrderUID, err)
		}

		for itemRows.Next() {
			var item model.Item
			if err := itemRows.Scan(
				&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name,
				&item.Sale, &item.Size, &item.TotalPrice, &item.NMID, &item.Brand, &item.Status,
			); err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("failed to scan item for order %v: %v", ord.OrderUID, err)
			}
			ord.Items = append(ord.Items, item)
		}
		itemRows.Close()

		ordersMap[ord.OrderUID] = ord
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %v", err)
	}

	return ordersMap, nil
}
