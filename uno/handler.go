package uno

import "github.com/glugox/unogo/context"

// HandlerFunc Handle the applicati
type HandlerFunc func(app *Application) Handler

// Closure Anonymous function, Used in Middleware Handler
type Closure func(req *context.Request) interface{}

// Handler Handler interface
type Handler interface {
	Process(request *context.Request, next Closure) interface{}
}
