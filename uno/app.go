package uno

import "github.com/glugox/unogo/log"

type Application struct {
	Env    string
	Debug  bool
	Logger log.Logger
}
