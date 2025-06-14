package main

import (
	"bank-api/internal/handlers"
	"bank-api/internal/services"
	"bank-api/pkg/database"
	"log"
	"os"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Не найден .env файл, используем переменные окружения")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL не установлен")
	}
	db, err := database.InitDB(dsn)
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET не установлен")
	}

	var (
		transactionService = services.NewTransactionService(db, jwtSecret)
		authService        = services.NewAuthService(db, jwtSecret)
		accountService     = services.NewAccountService(db, jwtSecret)
	)

	h := handlers.NewHandler(transactionService, authService, accountService)

	app := fiber.New(fiber.Config{
		ErrorHandler: h.ErrorHandler,
	})

	// Настройка CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000", // Укажите конкретный источник
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true, // Если вам нужно передавать куки
	}))

	cfg := swagger.Config{
		BasePath: "/",
		Path:     "swagger",
		Title:    "BankX API Docs",
	}

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(swagger.New(cfg))

	api := app.Group("/api")
	api.Post("/register", h.Register)
	api.Post("/login", h.Login)

	protected := api.Group("/", h.AuthMiddleware)
	protected.Get("/accounts", h.GetAccounts)
	protected.Post("/transfer", h.Transfer)
	protected.Post("/deposit/:id", h.Deposit)
	protected.Post("/withdraw/:id", h.Withdraw)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("Сервер запущен на порту %s", port)
	log.Fatal(app.Listen(":" + port))
}
