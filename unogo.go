package unogo

import (
	"fmt"

	"github.com/glugox/unogo/log"
	"github.com/glugox/unogo/log/record"
	"github.com/glugox/unogo/uno"
)

type Uno struct {
	App *uno.Application
	// handlers []uno.HandlerFunc
}

func SetupLogger() *log.Logger {
	l := log.NewLogger("default", record.DEBUG)
	l.PushDefaultHandler()
	return l
}

// New Create The Application
func New() *Uno {
	application := uno.NewApplication()
	application.Logger = log.NewLogger("local", record.DEBUG)
	t := &Uno{
		App: application,
	}
	//t.bootView()
	t.bootRoute()
	return t
}

func (th *Uno) bootRoute() {
	//r := router.New()
	//r.Statics(config.Route.Static)
	//th.App.RegisterRoute(r)

	fmt.Println("Boot router!")
}

// Run thinkgo application.
// Run() default run on HttpPort
// Run("localhost")
// Run(":1983")
// Run("127.0.0.1:1983")
func (th *Uno) Run(params ...string) {
	fmt.Println("Running on port :1983...")
}
