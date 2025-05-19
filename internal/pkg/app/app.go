package app

import (
	endpoint "agregator/api/internal/endpoint/app"
	api "agregator/api/internal/transport/rest"
)

type App struct {
	app *endpoint.App
	api *api.API
}

func New() *App {
	return &App{
		app: endpoint.New(),
		api: api.New(),
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
