package app

import (
	endpoint "agregator/api/internal/endpoint/app"
	"agregator/api/internal/interfaces"
	api "agregator/api/internal/transport/rest"
	"log"
)

type App struct {
	app *endpoint.App
	api *api.API
}

func New(logger interfaces.Logger) *App {
	api, err := api.New(logger)
	if err != nil {
		log.Fatal(err)
	}
	return &App{
		app: endpoint.New(),
		api: api,
	}
}

func (a *App) Run() {
	a.app.GetAPI("/ping", a.api.Check)
	a.app.GetV1("/max", a.api.GetMax)
	a.app.GetV1("/get/all", a.api.Get)
	a.app.GetV1("/get/top", a.api.GetTop)
	a.app.GetV1("/get/reg", a.api.GetRT)
	a.app.GetV1("/get/similar/:id", a.api.GetSimilar)
	a.app.GetV1("/get/:id", a.api.GetByID)
	a.app.Run(":8080")
}
