package main

import (
	"encoding/json"
	"errors"
	// "fmt"
	"io"
	"strings"

	// "fmt"
	// "io"
	"net/http"
	"net/url"
	"os"
	"time"

	// "github.com/gofiber/fiber/v2"
	// "github.com/gofiber/fiber/v2/middleware/logger"
	// "github.com/gofiber/fiber/v2/middleware/proxy"

	"github.com/abrander/ginproxy"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

// This service will be responsible for periodically caching
// the data from the Unity Publisher Administation API.

// TODO rename this API-Gateway?
// API gateway should first check the cache, otherwise the API will be called

type user struct {
	Email       string
	PublisherId string
}

func main() {
	r := gin.Default()

	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "unity-publisher-management",
		Key:         []byte("my temporary private secret key"),
		Timeout:     time.Hour * 72, // 3 days
		SendCookie:  true,
		TokenLookup: "header:Authorization,cookie:jwt",
		Authenticator: func(c *gin.Context) (interface{}, error) {
			url := createApiServiceUrl("/authenticate")
			res, err := http.Post(url, "application/json", c.Request.Body)
			if err != nil || res.StatusCode != http.StatusOK {
				return nil, jwt.ErrFailedAuthentication
			}

			var u user
			err = json.NewDecoder(res.Body).Decode(&u)
			if err != nil {
				return nil, jwt.ErrFailedAuthentication
			}
			return u, nil
		},
	})
	if err != nil {
		panic(err)
	}

	r.POST("/authenticate", authMiddleware.LoginHandler)

	proxy, _ := ginproxy.NewGinProxy("http://" + getApiServiceHost())

	// Automatically proxy all api requests to API service
	authGroup := r.Group("/api")
	authGroup.Use(authMiddleware.MiddlewareFunc())

	authGroup.Any("*any", func(c *gin.Context) {
		path := c.Param("any")

		// Check cache for sales API
		if strings.HasPrefix(path, "/sales") {
			println("sales path", path)
			err := fetchSalesFromCache(path, c)
			if err == nil {
				return
			}
		}

		println("Proxying request to API service...")
		proxy.Handler(c)
	})

	// TODO special handling for cached requests
	// authGroup.GET("/sales/:publisher/:month", func(c *gin.Context) {
	// 	publisher := c.Param("publisher")
	// 	month := c.Param("month")

	// 	println("Fetching sales for", publisher, "in", month, "...")

	// 	// Check the caching service first, then forward to API service if not found
	// 	cacheUrl := fmt.Sprintf("http://localhost:8082/sales/%s/%s", publisher, month)
	// 	res, err := http.Get(cacheUrl)
	// 	if err != nil {
	// 		println("Failed to fetch sales", err)
	// 	}
	// 	if res.StatusCode == http.StatusOK {
	// 		println("Retrieved sales from cache")

	// 		defer res.Body.Close()
	// 		body, _ := io.ReadAll(res.Body)
	// 		c.Data(http.StatusOK, "application/json", body)
	// 		return
	// 	}

	// 	println("Sales not found in cache, fetching from API service...")
	// 	// path := c.Request.URL.Path
	// 	// url := createApiServiceUrl(path)
	// 	proxy.Handler(c)
	// })

	r.Run(":8080")

	// app := fiber.New()
	// app.Post("/authenticate", proxy.Forward(createApiServiceUrl("/authenticate")))

	// app.Get("/api/sales/:publisher/:month", func(c *fiber.Ctx) error {
	// 	publisher := c.Params("publisher")
	// 	month := c.Params("month")

	// 	println("Fetching sales for", publisher, "in", month, "...")

	// 	// Check the caching service first, then forward to API service if not found
	// 	cacheUrl := fmt.Sprintf("http://localhost:8082/sales/%s/%s", publisher, month)
	// 	res, err := http.Get(cacheUrl)
	// 	if err != nil {
	// 		println("Failed to fetch sales", err)
	// 	}
	// 	if res.StatusCode == http.StatusOK {
	// 		println("Retrieved sales from cache")

	// 		defer res.Body.Close()
	// 		body, _ := io.ReadAll(res.Body)
	// 		return c.SendString(string(body))
	// 	}

	// 	println("Sales not found in cache, fetching from API service...")
	// 	path := c.Path()
	// 	url := createApiServiceUrl(path)
	// 	return proxy.Do(c, url)
	// })

	// app.All("/api/*", func(c *fiber.Ctx) error {
	// 	path := c.Path()
	// 	url := createApiServiceUrl(path)
	// 	return proxy.Do(c, url)
	// })

	// app.Use(logger.New())
	// app.Listen(":8080")
}

func fetchSalesFromCache(path string, c *gin.Context) error {
	// Check the caching service first, then forward to API service if not found
	cacheUrl, _ := url.JoinPath("http://localhost:8082", path)
	// cacheUrl := fmt.Sprintf("http://localhost:8082%s", path)
	res, err := http.Get(cacheUrl)
	if err != nil {
		println("Failed to fetch sales")
		return err
	}
	if res.StatusCode == http.StatusOK {
		println("Retrieved sales from cache")

		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		c.Data(http.StatusOK, "application/json", body)
		return nil
	}
	return errors.New("sales not found in cache")
}

func createApiServiceUrl(path string) string {
	return "http://" + getApiServiceHost() + path
}

func getApiServiceHost() string {
	if host, found := os.LookupEnv("UPM_API_SERVICE"); found {
		return host
	}
	return "localhost:8081"
}
