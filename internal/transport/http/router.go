package transporthttp

import (
	"net/http"

	"github.com/gorilla/mux"

	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
)

func NewRouter(
	taskHandler *httphandlers.TaskHandler,
	recurrenceHandler *httphandlers.RecurrenceHandler,
	docsHandler *swaggerdocs.Handler,
) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	// swagger
	router.HandleFunc("/swagger/openapi.json", docsHandler.ServeSpec).Methods(http.MethodGet)
	router.HandleFunc("/swagger/", docsHandler.ServeUI).Methods(http.MethodGet)
	router.HandleFunc("/swagger", docsHandler.RedirectToUI).Methods(http.MethodGet)

	// api
	api := router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/tasks", taskHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/tasks", taskHandler.List).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Update).Methods(http.MethodPut)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Delete).Methods(http.MethodDelete)

	api.HandleFunc("/tasks/{id:[0-9]+}/recurrence", recurrenceHandler.SetRule).Methods(http.MethodPut)
	api.HandleFunc("/tasks/{id:[0-9]+}/recurrence", recurrenceHandler.GetRule).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}/recurrence", recurrenceHandler.DeleteRule).Methods(http.MethodDelete)
	api.HandleFunc("/tasks/{id:[0-9]+}/recurrence/generate", recurrenceHandler.Generate).Methods(http.MethodPost)

	return router
}