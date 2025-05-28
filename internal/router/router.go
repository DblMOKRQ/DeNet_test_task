package router

import (
	"net/http"
	"strings"

	"github.com/DblMOKRQ/DeNet_test_task/internal/router/handlers"
	"github.com/DblMOKRQ/DeNet_test_task/internal/router/middleware"
	"github.com/DblMOKRQ/DeNet_test_task/pkg/jwt"
)

// Router представляет HTTP-роутер
type Router struct {
	mux         *http.ServeMux
	jwtService  *jwt.Service
	userHandler *handlers.UserHandler
}

// NewRouter создает новый экземпляр Router
func NewRouter(jwtService *jwt.Service, userHandler *handlers.UserHandler) *Router {
	return &Router{
		mux:         http.NewServeMux(),
		jwtService:  jwtService,
		userHandler: userHandler,
	}
}

// Setup настраивает маршруты
func (r *Router) Setup() http.Handler {
	// Middleware для всех запросов
	commonMiddleware := middleware.Chain(
		http.HandlerFunc(r.routeRequest),
		middleware.Logger,
		middleware.Recover,
		middleware.ContentTypeJSON,
	)

	// Middleware для защищенных маршрутов
	authMiddleware := middleware.Chain(
		commonMiddleware,
		middleware.JWTAuth(r.jwtService),
	)

	// Настройка маршрутов
	r.mux.Handle("/users/", authMiddleware)

	return r.mux
}

// routeRequest обрабатывает запросы и направляет их соответствующим обработчикам
func (r *Router) routeRequest(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method

	// Маршрут GET /users/{id}/status
	if method == http.MethodGet && matchPath(path, "/users/", "/status") {
		r.userHandler.GetUserStatus(w, req)
		return
	}

	// Маршрут GET /users/leaderboard
	if method == http.MethodGet && path == "/users/leaderboard" {
		r.userHandler.GetLeaderboard(w, req)
		return
	}

	// Маршрут POST /users/{id}/task/complete
	if method == http.MethodPost && matchPath(path, "/users/", "/task/complete") {
		r.userHandler.CompleteTask(w, req)
		return
	}

	// Маршрут POST /users/{id}/referrer
	if method == http.MethodPost && matchPath(path, "/users/", "/referrer") {
		r.userHandler.AddReferrer(w, req)
		return
	}

	// Если маршрут не найден
	http.NotFound(w, req)
}

// matchPath проверяет, соответствует ли путь шаблону prefix + "{id}" + suffix
func matchPath(path, prefix, suffix string) bool {
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	path = path[len(prefix):]

	if !strings.HasSuffix(path, suffix) {
		return false
	}

	id := path[:len(path)-len(suffix)]
	return id != "" && !strings.Contains(id, "/")
}
