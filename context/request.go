package context

import "net/http"

//Request HTTP request
type Request struct {
	Request       *http.Request
	method        string
	path          string
	query         map[string]string
	post          map[string]string
	files         map[string]*File
	session       Session
	CookieHandler *Cookie
}
