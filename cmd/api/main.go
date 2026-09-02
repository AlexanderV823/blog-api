package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"blog-api/internal/handler"
	"blog-api/internal/middleware"
	"blog-api/internal/repository"
	"blog-api/internal/service"
	"blog-api/pkg/auth"
	"blog-api/pkg/database"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

func main() {
	_ = pq.Driver{}

	// Инициализируем вывод логов параллельно в файл и консоль
	logFile, err := os.OpenFile("api.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Объединяем терминал и файл. Перенастраиваем стандартный логгер Go
	logOutput := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(logOutput)
	log.SetFlags(log.LstdFlags)

	// Создаем выделенный логгер для слоя Middleware
	loggerInstance := log.New(logOutput, "", log.LstdFlags)

	// Загружаем конфигурацию из .env файла
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	// Загружаем конфигурацию (секреты проверяются на обязательное наличие)
	cfg := loadConfig()

	// Подключаемся к базе данных PostgreSQL
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверяем соединение с БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to the database")

	// Вызываем функцию миграций для синхронизации структуры таблиц при каждом запуске
	log.Println("Running database migrations...")
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Critical error executing database migrations: %v", err)
	}
	log.Println("Database migrations applied successfully")

	// Инициализируем JWT менеджер
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiryHours)

	// Инициализируем слой репозиториев (Repository)
	userRepo := repository.NewUserRepo(db)
	postRepo := repository.NewPostRepo(db)
	commentRepo := repository.NewCommentRepo(db)

	// Инициализируем слой бизнес-логики (Service)
	userService := service.NewUserService(userRepo, jwtManager)
	postService := service.NewPostService(postRepo, userRepo)
	commentService := service.NewCommentService(commentRepo, postRepo, userRepo)

	// Инициализируем слой обработчиков (Handler)
	userHandler := handler.NewAuthHandler(userService)
	postHandler := handler.NewPostHandler(postService)
	commentHandler := handler.NewCommentHandler(commentService)

	// Инициализируем Middleware авторизации
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	// Явно инициализируем новый потокобезопасный RateLimiter (100 запросов в минуту)
	rateLimiterMiddleware := middleware.NewRateLimiter(100, time.Minute)

	// Инициализируем кастомный слой инфраструктурных Middleware с настроенным логгером
	customMiddleware := middleware.NewLoggingMiddleware(loggerInstance)

	// Настраиваем маршруты роутера
	router := chi.NewRouter()

	// Настраиваем кастомные глобальные middleware в строгой последовательности
	router.Use(customMiddleware.Recovery)  // Перехват паник
	router.Use(customMiddleware.RequestID) // Трассировка (Request ID)
	router.Use(customMiddleware.CORS)      // Обработка CORS заголовков и OPTIONS

	// Подключаем новый безопасный лимитер вместо customMiddleware.RateLimiter
	router.Use(rateLimiterMiddleware.Limit) // Защита от DDoS (Token Bucket с изоляцией мьютексов)

	router.Use(customMiddleware.Logger) // Логирование запросов

	// Роуты API
	router.Route("/api", func(r chi.Router) {
		// Публичные эндпоинты
		r.Post("/register", userHandler.Register)
		r.Post("/login", userHandler.Login)

		r.Get("/posts", postHandler.GetAll)
		r.Get("/posts/{id}", postHandler.GetByID)
		r.Get("/posts/{id}/comments", commentHandler.GetByPost)

		// Защищенные эндпоинты (требуют JWT)
		r.Group(func(protected chi.Router) {
			protected.Use(authMiddleware.RequireAuth)

			protected.Post("/posts", postHandler.Create)
			protected.Put("/posts/{id}", postHandler.Update)
			protected.Delete("/posts/{id}", postHandler.Delete)

			protected.Post("/posts/{id}/comments", commentHandler.CreateComment)
			protected.Put("/comments/{id}", commentHandler.Update)
			protected.Delete("/comments/{id}", commentHandler.Delete)
		})

		// Health check эндпоинт
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")

			// Создаем короткий контекст с тайм-аутом 2 секунды, чтобы не вешать запрос
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			// Выполняем Ping и тестовый легковесный запрос к одной из таблиц схемы
			// для проверки успешного прохождения миграций
			err := db.PingContext(ctx)
			if err == nil {
				_, err = db.ExecContext(ctx, "SELECT 1 FROM users LIMIT 1;")
			}

			if err != nil {
				// Записываем детальную ошибку драйвера СУБД в защищенный лог сервера
				log.Printf("[ERROR] Health check database failure: %v", err)

				// Если БД недоступна или схема не инициализирована, отдаем 503 Service Unavailable
				w.WriteHeader(http.StatusServiceUnavailable)

				// Используем безопасный fmt.Appendf с нейтральной заглушкой без раскрытия деталей
				res := fmt.Appendf(nil, `{"status":"down","error":"database connection failed","service":"blog-api"}`)
				_, _ = w.Write(res)
				return
			}

			// Если всё в порядке, отдаем стабильный 200 OK
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","database":"connected","service":"blog-api"}`))
		})
	})

	// Раздача главной страницы
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	})

	// Изолированная раздача статических ресурсов (CSS, JS)
	staticServer := http.FileServer(http.Dir("web"))
	router.Handle("/static/*", http.StripPrefix("/static/", staticServer))

	// Запуск HTTP сервера
	serverAddr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	log.Printf("Starting HTTP server on %s", serverAddr)

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server failed to start: %v", err)
	}
}

// Config представляет конфигурацию приложения
type Config struct {
	// Server
	ServerHost string
	ServerPort int

	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// JWT
	JWTSecret      string
	JWTExpiryHours int

	// Cache
	CacheTTLMinutes int
}

// loadConfig загружает конфигурацию из переменных окружения
func loadConfig() *Config {
	return &Config{
		DBSSLMode:       getEnv("DB_SSL_MODE", "disable"),
		CacheTTLMinutes: getEnvAsInt("CACHE_TTL_MINUTES", 15),

		ServerHost: getEnvRequired("SERVER_HOST"),
		ServerPort: getEnvAsIntRequired("SERVER_PORT"),

		DBHost:     getEnvRequired("DB_HOST"),
		DBPort:     getEnvAsIntRequired("DB_PORT"),
		DBName:     getEnvRequired("DB_NAME"),
		DBUser:     getEnvRequired("DB_USER"),
		DBPassword: getEnvRequired("DB_PASSWORD"),

		JWTSecret:      getEnvRequired("JWT_SECRET"),
		JWTExpiryHours: getEnvAsIntRequired("JWT_EXPIRY_HOURS"),
	}
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvRequired принудительно требует наличие переменной окружения, иначе аварийно завершает работу
func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Critical environment variable missing: %s. Please check your .env file.", key)
	}
	return value
}

// getEnvAsInt получает значение переменной окружения как int или возвращает значение по умолчанию
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getEnvAsIntRequired принудительно требует числовую переменную, иначе завершает работу
func getEnvAsIntRequired(key string) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		log.Fatalf("Critical environment variable missing: %s", key)
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Fatalf("Invalid integer format for environment variable %s: %v", key, err)
	}
	return value
}
