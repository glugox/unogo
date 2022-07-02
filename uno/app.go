package uno

import "github.com/glugox/unogo/log"

type Application struct {
	Env    string
	Debug  bool
	Logger *log.Logger
}

// Creates new Applications
func NewApplication() *Application {
	return &Application{Env: "local"}
}
