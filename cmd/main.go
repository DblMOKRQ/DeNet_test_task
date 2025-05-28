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

	log, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	// Инициализация репозитория
	repo, err := postgres.NewRepository(cfg.Storage.User, cfg.Storage.Password, cfg.Storage.DBName, cfg.Storage.Sslmode)
	if err != nil {
		log.Fatal("Failed to initialize repository: ", zap.Error(err))
	}
	defer repo.Close()

	// Инициализация сервисов
	jwtService := jwt.NewService(cfg.JWT.SecretKey, cfg.JWT.TokenDuration)
	userService := service.NewUserService(repo)

	// Инициализация обработчиков
	userHandler := handlers.NewUserHandler(userService)

	// Инициализация роутера
	r := router.NewRouter(jwtService, userHandler)
	handler := r.Setup()

	addr := cfg.Rest.Host + ":" + cfg.Rest.Port
	// Инициализация HTTP сервера
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Запуск сервера в горутине
	go func() {
		log.Info("Starting server ", zap.String("addr: ", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server: %v", zap.Error(err))
		}
	}()

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", zap.Error(err))
	}

	log.Info("Server exited properly")
}
