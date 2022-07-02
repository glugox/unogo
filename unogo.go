package unogo

import (
	"github.com/glugox/unogo/log"
	"github.com/glugox/unogo/log/record"
)

type Unogo struct {
	App      *unogo.Application
	handlers []unogo.HandlerFunc
}

func SetupLogger() *log.Logger {
	l := log.NewLogger("default", record.DEBUG)
	l.PushDefaultHandler()
	return l
}
