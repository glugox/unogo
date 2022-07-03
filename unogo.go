package unogo

import (
	"fmt"
	"net/http"
	"time"

	"github.com/glugox/unogo/config"
	"github.com/glugox/unogo/helper"
	"github.com/glugox/unogo/log"
	"github.com/glugox/unogo/log/record"
	"github.com/glugox/unogo/orm"
	"github.com/glugox/unogo/router"
	"github.com/glugox/unogo/uno"
)

// registerRouteFunc
type registerRouteFunc func(route *router.Route)

// registerConfigFunc
type registerConfigFunc func()

// Uno
type Uno struct {
	App      *uno.Application
	handlers []uno.HandlerFunc
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

	// Configure DB connection
	config := orm.Config{
		Driver: "mysql",
		Dsn:    "root:root@tcp(127.0.0.1:3306)/unogo?charset=utf8&parseTime=true",
	}
	db, _ := orm.Open(config)

	application.DB = db
	t := &Uno{
		App: application,
	}
	//t.bootView()
	t.bootRoute()
	return t
}

// RegisterRoute Register Route
func (th *Uno) RegisterRoute(register registerRouteFunc) {
	route := th.App.GetRoute()
	defer route.Register()
	register(route)
}

func (th *Uno) bootRoute() {
	r := router.New()
	r.Statics(config.Route.Static)
	th.App.RegisterRoute(r)

	fmt.Println("Boot router!")
}

// RegisterConfig Register Config
func (uno *Uno) RegisterHandler(handler uno.HandlerFunc) {
	uno.handlers = append(uno.handlers, handler)
}

// Run thinkgo application.
// Run() default run on HttpPort
// Run("localhost")
// Run(":1983")
// Run("127.0.0.1:1983")
func (u *Uno) Run(params ...string) {
	var err error
	var endRunning = make(chan bool, 1)
	var addrs = helper.ParseAddr(params...)

	u.RegisterHandler(uno.NewKernel)

	pipeline := NewPipeline()
	for _, h := range u.handlers {
		pipeline.Pipe(h(u.App))
	}

	go func() {
		u.App.Logger.Debug("Uno server running on http://%s", addrs)
		err = http.ListenAndServe(addrs, pipeline)
		if err != nil {
			fmt.Println(err.Error())
			time.Sleep(100 * time.Microsecond)
			endRunning <- true
		}
	}()

	<-endRunning

	fmt.Println("Running on port :1983...")
}
