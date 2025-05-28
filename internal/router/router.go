package router

import (
	"net/http"

	"github.com/DblMOKRQ/DeNet_test_task/internal/router/handlers"
	"github.com/DblMOKRQ/DeNet_test_task/internal/router/middleware"
	"github.com/DblMOKRQ/DeNet_test_task/pkg/jwt"
	"go.uber.org/zap"
)

// Router обрабатывает HTTP запросы
type Router struct {
	jwtService  *jwt.Service
	userHandler *handlers.UserHandler
	log         *zap.Logger
}

// NewRouter создает новый экземпляр Router
func NewRouter(jwtService *jwt.Service, userHandler *handlers.UserHandler, log *zap.Logger) *Router {
	return &Router{
		jwtService:  jwtService,
		userHandler: userHandler,
		log:         log.Named("router"),
	}
}

// Setup настраивает маршруты и middleware
func (r *Router) Setup() http.Handler {
	// Создание маршрутизатора
	mux := http.NewServeMux()

	// Регистрация обработчиков
	mux.Handle("/users/register",
		middleware.Chain(
			http.HandlerFunc(r.userHandler.LoginUser),
			middleware.Recover(r.log),
			middleware.Logger(r.log),
			middleware.ContentTypeJSON,
		),
	)

	// Для всех остальных маршрутов применяем JWT middleware
	protected := http.NewServeMux()
	protected.HandleFunc("/users/leaderboard", r.userHandler.GetLeaderboard)
	protected.HandleFunc("/users/status", r.userHandler.GetUserStatus)
	protected.HandleFunc("/users/task/complete", r.userHandler.CompleteTask)
	protected.HandleFunc("/users/referrer", r.userHandler.AddReferrer)

	// Применение middleware к защищенным маршрутам
	protectedHandler := middleware.Chain(
		protected,
		middleware.Recover(r.log),
		middleware.Logger(r.log),
		middleware.JWTAuth(r.jwtService, r.log),
		middleware.ContentTypeJSON,
	)

	// Объединяем защищенные и публичные маршруты
	mux.Handle("/", protectedHandler)

	return mux
}
