package main

import (
	"golang-api/config"
	"golang-api/routes"
	"log"
	"net/http"
)

func main() {
    // Initialize database connections
    config.ConnectMongoDB()
    config.ConnectRedis()

    // Setup routes
    r := routes.SetupRoutes()

    // Start the server
    log.Fatal(http.ListenAndServe(":8000", r))
}
