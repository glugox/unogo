package uno

import (
	"github.com/glugox/unogo/log"
	"github.com/glugox/unogo/orm"
	"github.com/glugox/unogo/router"
)

type Application struct {
	Env    string
	Debug  bool
	Logger *log.Logger
	DB     *orm.Connection
	route  *router.Route
}

// Creates new Applications
func NewApplication() *Application {
	return &Application{Env: "local"}
}

// RegisterRoute Register Route for Application
func (a *Application) RegisterRoute(r *router.Route) {
	a.route = r
}

// GetRoute Get the router of the application
func (a *Application) GetRoute() *router.Route {
	return a.route
}
