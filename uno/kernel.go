package uno

import (
	"github.com/glugox/unogo/context"
	"github.com/glugox/unogo/router"
)

type Kernel struct {
	Route *router.Route
}

// NewKernel The default type
func NewKernel(app *Application) Handler {
	return &Kernel{
		Route: app.GetRoute(),
	}
}

// Process Process the request to a router and return the response.
func (h *Kernel) Process(request *context.Request, next Closure) interface{} {
	rule, err := h.Route.Dispatch(request)

	if err != nil {
		return context.NotFoundResponse()
	}

	return router.RunRoute(request, rule)
}
