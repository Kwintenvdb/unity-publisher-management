package main

import (
	// "context"
	"io"
	"net/http"
	// "net/url"
	"os"

	// "github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/proxy"
)

// This service will be responsible for periodically caching
// the data from the Unity Publisher Administation API.

// TODO rename this API-Gateway?
// API gateway should first check the cache, otherwise the API will be called

func main() {
	// rdb := redis.NewClient(&redis.Options{
	// 	Addr:     "localhost:6379",
	// 	Password: "", // no password set
	// 	DB:       0,  // use default DB
	// })
	// println("Connected to Redis")

	// ctx := context.Background()
	// rdb.Set(ctx, "my_key", "my_value", 0)
	
	// value := rdb.Get(ctx, "my_key").Val()
	// println(value)

	// TODO proxy middleware?
	app := fiber.New()
	// app.Post("/authenticate", authenticate)
	app.Post("/authenticate", proxy.Forward(createApiServiceUrl("/authenticate")))
	app.Get("/sales/:month", fetchSales)

	// sales := fetchSales()
	// rdb.Set(ctx, "sales/202210", sales, 0)

	// s := rdb.Get(ctx, "sales/202210").Val()
	// println(s)

	app.Use(logger.New())
	app.Listen(":8080")
}

func fetchSales(c *fiber.Ctx) error {
	
	println("request path: " + c.Path())
	res, err := http.Get(createApiServiceUrl(c.Path()))
	if err != nil {
		return err
	}

	defer res.Body.Close()
	sales, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	c.Send(sales)
	return nil
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
