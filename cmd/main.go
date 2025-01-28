// Path: cmd/main.go
package main

import (
	"bank-api/internal/handlers"
	"bank-api/internal/services"
	"bank-api/pkg/database"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, proceeding with environment variables")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "bank.db"
	}
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set in environment variables")
	}

	// Use interfaces for service dependencies.
	var (
		transactionService services.TransactionService = services.NewTransactionService(db, jwtSecret)
		authService        services.AuthService        = services.NewAuthService(db, jwtSecret)
		accountService     services.AccountService     = services.NewAccountService(db, jwtSecret)
	)

	h := handlers.NewHandler(transactionService, authService, accountService)

	app := fiber.New(fiber.Config{
		ErrorHandler: h.ErrorHandler,
	})

	// CORS middleware configuration
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:8080", // Укажите здесь URL вашего фронтенда
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	app.Use(recover.New())
	app.Use(logger.New())

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
	log.Printf("Server is running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
