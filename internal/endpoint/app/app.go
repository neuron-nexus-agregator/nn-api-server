package app

import (
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
