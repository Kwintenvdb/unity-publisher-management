package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/proxy"
)

// This service will be responsible for periodically caching
// the data from the Unity Publisher Administation API.

// TODO rename this API-Gateway?
// API gateway should first check the cache, otherwise the API will be called

func main() {
	app := fiber.New()
	app.Post("/authenticate", proxy.Forward(createApiServiceUrl("/authenticate")))
	app.All("/api/*", func(c *fiber.Ctx) error {
		path := c.Path()
		url := createApiServiceUrl(path)
		return proxy.Do(c, url)
	})

	app.Use(logger.New())
	app.Listen(":8080")
}

func createApiServiceUrl(path string) string {
	return "http://" + getApiServiceHost() + path
}

func getApiServiceHost() string {
	if host, found := os.LookupEnv("UPM_API_SERVICE_HOST"); found {
		return host
	}
	return "localhost:8081"
}
