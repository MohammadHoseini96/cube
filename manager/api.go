package manager

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type ErrResponse struct {
	HttpStatusCode int
	Message        string
}

type Api struct {
	Address string
	Port    int
	Manager *Manager
	Router  *chi.Mux
}

func (a *Api) initRouter() {
	a.Router = chi.NewRouter()
	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.GetTasksHandler)
		r.Route("/{taskID}", func(r chi.Router) {
			r.Delete("/", a.StopTaskHandler)
		})
	})
}

func (a *Api) Star() {
	a.initRouter()
	http.ListenAndServe(fmt.Sprintf("%s:%d", a.Address, a.Port), a.Router)
}