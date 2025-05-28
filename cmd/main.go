package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DblMOKRQ/DeNet_test_task/internal/config"
	"github.com/DblMOKRQ/DeNet_test_task/internal/repository/postgres"
	"github.com/DblMOKRQ/DeNet_test_task/internal/router"
	"github.com/DblMOKRQ/DeNet_test_task/internal/router/handlers"
	"github.com/DblMOKRQ/DeNet_test_task/internal/service"
	"github.com/DblMOKRQ/DeNet_test_task/pkg/jwt"
	"github.com/DblMOKRQ/DeNet_test_task/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Загрузка конфигурации
	cfg := config.MustLoad()

	// Инициализация логгера
	log, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("Starting application",
		zap.String("version", "1.0.0"))

	// Инициализация репозитория
	log.Info("Initializing repository")
	repo, err := postgres.NewRepository(
		cfg.Storage.User,
		cfg.Storage.Password,
		cfg.Storage.Host,
		cfg.Storage.Port,
		cfg.Storage.DBName,
		cfg.Storage.Sslmode,
		log,
	)
	if err != nil {
		log.Fatal("Failed to initialize repository", zap.Error(err))
	}
	defer repo.Close()

	// Инициализация сервисов
	log.Info("Initializing services")

	jwtService := jwt.NewService(cfg.JWT.SecretKey, cfg.JWT.TokenDuration, log)
	userService := service.NewUserService(repo, log)

	// Инициализация обработчиков
	log.Info("Initializing handlers")
	userHandler := handlers.NewUserHandler(userService, jwtService, log)

	// Инициализация роутера
	log.Info("Setting up router")
	r := router.NewRouter(jwtService, userHandler, log)
	handler := r.Setup()

	addr := cfg.Rest.Host + ":" + cfg.Rest.Port
	log.Info("Server address configured", zap.String("addr", addr))

	// Инициализация HTTP сервера
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запуск сервера в горутине
	go func() {
		log.Info("Starting server", zap.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info("Shutting down server", zap.String("signal", sig.String()))

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited properly")
}
