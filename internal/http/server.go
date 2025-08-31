package http

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "strconv"
    "time"

    "l0/internal/cache"
    "l0/internal/db"
)

type Server struct {
    cache *cache.Cache
    repo *db.OrderRepository
    server *http.Server
    logger *log.Logger
}

func NewServer(port int, cache *cache.Cache, repo *db.OrderRepository, logger *log.Logger) *Server {
    mux := http.NewServeMux()
    
    s := &Server{
        cache: cache,
        repo: repo,
        logger: logger,
    }
    
    mux.HandleFunc("/order/", s.handleGetOrder)
    mux.HandleFunc("/", s.handleRoot)
    
    s.server = &http.Server{
        Addr:         ":" + strconv.Itoa(port),
        Handler:      mux,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }
    
    return s
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "web/index.html")
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
    // Извлечение order_uid из URL
    orderUID := r.URL.Path[len("/order/"):]
    if orderUID == "" {
        http.Error(w, "Order UID is required", http.StatusBadRequest)
        return
    }
    
    s.logger.Printf("Получен запрос на заказ: %v", orderUID)
    
    // Попытка получить заказ из кэша (или БД, если в кэше нет)
    order, found := s.cache.GetOrder(orderUID)
    if !found {
        s.logger.Printf("Заказ %v не найден в кэше, запрашиваем в БД", orderUID)
        
        // Если нет в кэше, ищем в БД
        dbOrder, err := s.repo.GetOrderByID(r.Context(), orderUID)
        if err != nil {
            s.logger.Printf("Ошибка получения заказа %v из БД: %v", orderUID, err)
            http.Error(w, "Заказ не найден", http.StatusNotFound)
            return
        }
        
        // Добавляем в кэш для будущих запросов
        order = *dbOrder
        s.cache.SetOrder(order)
    }
    
    // Установка заголовков
    w.Header().Set("Content-Type", "application/json")
    
    // Отправка ответа
    if err := json.NewEncoder(w).Encode(order); err != nil {
        s.logger.Printf("Error encoding response: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    }
}

func (s *Server) Start() error {
    s.logger.Printf("Запуск сервера на порте %v", s.server.Addr)
    return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Println("Остановка сервера...")
    return s.server.Shutdown(ctx)
}