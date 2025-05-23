package app

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

type App struct {
	router *gin.Engine
	api    *gin.RouterGroup
	api_v1 *gin.RouterGroup
}

func New() *App {
	router := gin.Default()
	api := router.Group("/api")
	api_v1 := api.Group("/v1")

	allowedAddrStr := os.Getenv("ALLOWED_CORS_ORIGINS")
	allowedOrigins := strings.Split(allowedAddrStr, ",")
	for i, origin := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(origin)
	}

	config := cors.DefaultConfig()
	config.AllowOrigins = allowedOrigins // Используем массив доменов из переменной окружения
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "Cache-Control", "X-Requested-With"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	// add CORS middleware to api_v1
	api_v1.Use(cors.New(config))

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	return &App{
		router: router,
		api:    api,
		api_v1: api_v1,
	}
}

func (a *App) GetAPI(path string, fn gin.HandlerFunc) {
	a.api.GET(path, fn)
}

func (a *App) GetV1(path string, fn gin.HandlerFunc) {
	a.api_v1.GET(path, fn)
}

func (a *App) PostV1(path string, fn gin.HandlerFunc) {
	a.api_v1.POST(path, fn)
}

func (a *App) Run(addr string) {
	a.router.Run(addr)
}
