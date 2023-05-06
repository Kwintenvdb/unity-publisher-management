package main

import (
	"fmt"
	"io"
	"net/http"
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
	
	app.Get("/api/sales/:publisher/:month", func(c *fiber.Ctx) error {
		publisher := c.Params("publisher")
		month := c.Params("month")

		println("Fetching sales for", publisher, "in", month, "...")

		// Check the caching service first, then forward to API service if not found
		cacheUrl := fmt.Sprintf("http://localhost:8082/sales/%s/%s", publisher, month)
		res, err := http.Get(cacheUrl)
		if err != nil {
			println("Failed to fetch sales", err)
		}
		if res.StatusCode == http.StatusOK {
			println("Retrieved sales from cache")

			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)
			return c.SendString(string(body))
		}

		println("Sales not found in cache, fetching from API service...")
		path := c.Path()
		url := createApiServiceUrl(path)
		return proxy.Do(c, url)
	})

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
