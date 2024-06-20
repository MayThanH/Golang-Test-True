package routes

import (
	"golang-api/controllers"

	"github.com/gorilla/mux"
)

func SetupRoutes() *mux.Router {
    r := mux.NewRouter()
    r.HandleFunc("/items", controllers.CreateItem).Methods("POST")
    r.HandleFunc("/items/{id}", controllers.GetItem).Methods("GET")
    r.HandleFunc("/items/{id}", controllers.UpdateItem).Methods("PUT")
    r.HandleFunc("/items/{id}", controllers.DeleteItem).Methods("DELETE")
    r.HandleFunc("/items", controllers.GetAllItems).Methods("GET")
    return r
}
