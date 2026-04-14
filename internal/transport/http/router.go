package transporthttp

import (
	"net/http"

	"github.com/gorilla/mux"

	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
)

func NewRouter(taskHandler *handlers.TaskHandler, recurrenceHandler *handlers.RecurrenceHandler, docsHandler *swaggerdocs.Handler) *mux.Router {
    r := mux.NewRouter()
    api := r.PathPrefix("/api/v1").Subrouter()

    api.HandleFunc("/tasks",        taskHandler.Create).Methods(http.MethodPost)
    api.HandleFunc("/tasks",        taskHandler.List).Methods(http.MethodGet)
    api.HandleFunc("/tasks/{id}",   taskHandler.GetByID).Methods(http.MethodGet)
    api.HandleFunc("/tasks/{id}",   taskHandler.Update).Methods(http.MethodPut)
    api.HandleFunc("/tasks/{id}",   taskHandler.Delete).Methods(http.MethodDelete)

    // Recurrence
    api.HandleFunc("/tasks/{id}/recurrence",          recurrenceHandler.SetRule).Methods(http.MethodPut)
    api.HandleFunc("/tasks/{id}/recurrence",          recurrenceHandler.GetRule).Methods(http.MethodGet)
    api.HandleFunc("/tasks/{id}/recurrence",          recurrenceHandler.DeleteRule).Methods(http.MethodDelete)
    api.HandleFunc("/tasks/{id}/recurrence/generate", recurrenceHandler.Generate).Methods(http.MethodPost)

    r.PathPrefix("/swagger/").Handler(docsHandler)
    return r
}
