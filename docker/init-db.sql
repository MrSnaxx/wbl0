-- Создаем пользователя и БД
CREATE USER order_user WITH PASSWORD 'order_pass';
CREATE DATABASE order_db OWNER order_user;
GRANT ALL PRIVILEGES ON DATABASE order_db TO order_user;

-- Переключаемся на новую БД
\c order_db order_user

-- Создаем таблицы
-- Таблица orders
CREATE TABLE IF NOT EXISTS orders (
    order_uid varchar NOT NULL,
    track_number varchar NOT NULL,
    entry varchar NOT NULL,
    locale varchar NOT NULL,
    internal_signature varchar,
    customer_id varchar NOT NULL,
    delivery_service varchar NOT NULL,
    shardkey varchar,
    sm_id integer NOT NULL,
    date_created timestamptz DEFAULT CURRENT_TIMESTAMP,
    oof_shard varchar,
    CONSTRAINT orders_pkey PRIMARY KEY (order_uid)
);

-- Таблица delivery
CREATE TABLE IF NOT EXISTS delivery (
    id serial NOT NULL,
    order_uid varchar NOT NULL,
    name varchar,
    phone varchar,
    zip varchar NOT NULL,
    city varchar NOT NULL,
    address varchar NOT NULL,
    region varchar NOT NULL,
    email varchar,
    CONSTRAINT delivery_pkey PRIMARY KEY (id),
    CONSTRAINT delivery_order_uid_key UNIQUE (order_uid),
    CONSTRAINT delivery_order_uid_fkey FOREIGN KEY (order_uid)
        REFERENCES orders (order_uid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

-- Таблица items
CREATE TABLE IF NOT EXISTS items (
    chrt_id integer NOT NULL,
    track_number varchar NOT NULL,
    price numeric NOT NULL,
    rid varchar,
    name varchar NOT NULL,
    sale numeric,
    size varchar NOT NULL,
    total_price numeric NOT NULL,
    nm_id integer,
    brand varchar,
    status integer NOT NULL,
    CONSTRAINT items_pkey PRIMARY KEY (chrt_id)
);

-- Таблица order_items
CREATE TABLE IF NOT EXISTS order_items (
    order_uid varchar NOT NULL,
    chrt_id integer NOT NULL,
    CONSTRAINT order_items_pkey PRIMARY KEY (order_uid, chrt_id),
    CONSTRAINT order_items_chrt_id_fkey FOREIGN KEY (chrt_id)
        REFERENCES items (chrt_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE,
    CONSTRAINT order_items_order_uid_fkey FOREIGN KEY (order_uid)
        REFERENCES orders (order_uid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);

-- Таблица payments
CREATE TABLE IF NOT EXISTS payments (
    transaction varchar NOT NULL,
    order_uid varchar NOT NULL,
    request_id varchar,
    currency varchar NOT NULL,
    provider varchar,
    amount numeric NOT NULL,
    payment_dt bigint NOT NULL,
    bank varchar NOT NULL,
    delivery_cost numeric NOT NULL,
    goods_total integer NOT NULL,
    custom_fee numeric NOT NULL,
    CONSTRAINT payments_pkey PRIMARY KEY (transaction),
    CONSTRAINT payments_order_uid_fkey FOREIGN KEY (order_uid)
        REFERENCES orders (order_uid) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);