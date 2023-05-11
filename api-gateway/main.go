package main

import (
	// "encoding/json"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/abrander/ginproxy"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"

	"github.com/Kwintenvdb/unity-publisher-management/api-gateway/auth"
)

type user struct {
	Email       string
	PublisherId string
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func main() {
	r := gin.Default()

	proxy, _ := ginproxy.NewGinProxy("http://" + getApiServiceHost())

	// NOTE: We need to invalidate the token somehow / log out the user
	// if any of the Unity API endpoints return a 401
	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "unity-publisher-management",
		Key:         []byte("my temporary private secret key"),
		Timeout:     time.Hour * 72, // 3 days
		SendCookie:  true,
		TokenLookup: "header:Authorization,cookie:jwt",
		Authenticator: func(c *gin.Context) (interface{}, error) {
			writer := &responseBodyWriter{
				body:           &bytes.Buffer{},
				ResponseWriter: c.Writer,
			}
			c.Writer = writer

			proxy.Handler(c)

			if c.Writer.Status() != http.StatusOK {
				return nil, jwt.ErrFailedAuthentication
			}

			var u user
			err := json.NewDecoder(writer.body).Decode(&u)
			if err != nil {
				return nil, jwt.ErrFailedAuthentication
			}
			return u, nil
		},
		LoginResponse: func(c *gin.Context, code int, token string, expire time.Time) {
			user := c.MustGet("user").(*user)

			scheduleSalesCaching(c, user, token)

			c.JSON(http.StatusOK, gin.H{
				"email":       user.Email,
				"publisherId": user.PublisherId,
				"token":       token,
				"expire":      expire.Format(time.RFC3339),
			})
		},
	})
	if err != nil {
		panic(err)
	}

	r.POST("/authenticate", authMiddleware.LoginHandler)


	// Automatically proxy all api requests to API service
	authGroup := r.Group("/api")
	authGroup.Use(authMiddleware.MiddlewareFunc())

	authGroup.Any("*any", func(c *gin.Context) {
		path := c.Param("any")

		// Check cache for sales API
		if strings.HasPrefix(path, "/sales") {
			println("sales path", path)
			// Check the caching service first, then forward to API service if not found
			err := fetchSalesFromCache(path, c)
			if err == nil {
				return
			}
		}

		println("Proxying request to API service...")
		proxy.Handler(c)
	})

	r.Run(":8080")
}

func fetchSalesFromCache(path string, c *gin.Context) error {
	cacheUrl, _ := url.JoinPath("http://localhost:8082", path)
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

func scheduleSalesCaching(c *gin.Context, user *user, token string) {
	kharmaToken, _ := c.Cookie("kharma_token")
	kharmaSession, _ := c.Cookie("kharma_session")
	auth.SendUserAuthenticatedMessage(user.PublisherId, kharmaSession, kharmaToken, token)
}

func getApiServiceHost() string {
	if host, found := os.LookupEnv("UPM_API_SERVICE"); found {
		return host
	}
	return "localhost:8081"
}
