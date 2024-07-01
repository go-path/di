package controller

import (
	"net/http"

	"github.com/go-path/di"
)

type IndexController struct{}

func (s *IndexController) Path() string {
	return "/"
}

func (s *IndexController) Index(req *http.Request, w http.ResponseWriter) string {
	return "Hello World!"
}

func (s *IndexController) Ping(req *http.Request, w http.ResponseWriter) string {
	return "pong"
}

func init() {
	di.Register(&IndexController{}) // singleton
}
