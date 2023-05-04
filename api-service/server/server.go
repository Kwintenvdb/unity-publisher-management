package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Kwintenvdb/unity-publisher-management/api"
	"github.com/Kwintenvdb/unity-publisher-management/logger"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

type server struct {
	logger logger.Logger
}

type user struct {
	Email       string
	PublisherId string
}

func Start() {
	logger := logger.NewLogger()
	server := server{
		logger: logger,
	}

	r := gin.Default()

	// NOTE: We need to invalidate the token somehow / log out the user
	// if any of the Unity API endpoints return a 401
	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "unity-publisher-management",
		Key:         []byte("my temporary private secret key"),
		Timeout:     time.Hour * 72, // 3 days
		SendCookie:  true,
		TokenLookup: "header:Authorization,cookie:jwt",
		Authenticator: func(c *gin.Context) (interface{}, error) {
			email, publisher, err := server.authenticate(c)
			if err != nil {
				return nil, jwt.ErrFailedAuthentication
			}
			user := &user{
				Email:       email,
				PublisherId: publisher,
			}
			c.Set("user", user)
			return user, nil
		},
		LoginResponse: func(c *gin.Context, code int, token string, expire time.Time) {
			user := c.MustGet("user").(*user)

			kharmaToken, _ := c.Cookie("kharma_token")
			kharmaSession, _ := c.Cookie("kharma_session")

			// Inform the scheduling service
			schedulingPayload := fmt.Sprintf(`{
				"publisher": "%s",
				"kharmaSession": "%s",
				"kharmaToken": "%s",
				"jwt": "%s"
			}`, user.PublisherId, kharmaSession, kharmaToken, token)
			_, err := http.Post("http://localhost:8083/schedule", "application/json", bytes.NewReader([]byte(schedulingPayload)))
			if err != nil {
				logger.Warnw("Failed to schedule sales fetching", "error", err)
			}

			c.JSON(http.StatusOK, gin.H{
				"email":       user.Email,
				"publisherId": user.PublisherId,
				"token":       token,
				"expire":      expire.Format(time.RFC3339),
			})
		},
	})

	if err != nil {
		logger.Fatalw("Failed to create auth middleware", "error", err)
	}

	r.POST("/authenticate", authMiddleware.LoginHandler)

	auth := r.Group("/api")
	auth.Use(authMiddleware.MiddlewareFunc())

	auth.GET("/sales/:publisher/:month", server.fetchSales)
	auth.GET("/months/:publisher", server.fetchMonths)
	auth.GET("/packages", server.fetchPackages)

	logger.Info("Starting server on port 8081")
	r.Run(":8081")
}

func (s *server) authenticate(c *gin.Context) (string, string, error) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	if len(email) == 0 || len(password) == 0 {
		c.String(http.StatusBadRequest, "Missing email or password")
		return "", "", errors.New("missing email or password")
	}

	apiClient := api.NewClient(s.logger)
	authResponse, err := apiClient.Authenticate(email, password)
	if err != nil {
		c.String(http.StatusUnauthorized, "Failed to authenticate")
		return "", "", err
	}

	c.SetCookie("kharma_token", authResponse.KharmaToken, 0, "", "", false, true)
	c.SetCookie("kharma_session", authResponse.KharmaSession, 0, "", "", false, true)

	return email, authResponse.PublisherId, nil
}

func (s *server) fetchSales(c *gin.Context) {
	token, session, err := getSessionData(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Failed to authenticate")
		return
	}

	publisher := c.Param("publisher")
	month := c.Param("month")

	cacheUrl := fmt.Sprintf("http://localhost:8082/sales/%s/%s", publisher, month)
	res, err := http.Get(cacheUrl)
	if err == nil && res.StatusCode == http.StatusOK {
		s.logger.Debug("Retrieved sales from cache")
		c.DataFromReader(http.StatusOK, res.ContentLength, "application/json", res.Body, nil)
		return
	} else {
		s.logger.Debug("Sales not found in cache")
	}

	apiClient := api.NewClient(s.logger)
	sales, err := apiClient.FetchSales(publisher, month, token, session)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch sales")
		return
	}

	// Cache the sales
	s.logger.Debug("Sales retrieved. Caching sales...")
	salesData, _ := json.Marshal(sales)
	_, err = http.Post(cacheUrl, "application/json", bytes.NewReader(salesData))
	if err != nil {
		s.logger.Warnw("Failed to cache sales", "error", err)
	}

	c.JSON(http.StatusOK, sales)
}

func (s *server) fetchMonths(c *gin.Context) {
	token, session, err := getSessionData(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Failed to authenticate")
		return
	}

	publisher := c.Param("publisher")

	apiClient := api.NewClient(s.logger)
	months, err := apiClient.FetchMonths(publisher, token, session)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch months")
		return
	}
	c.JSON(http.StatusOK, months)
}

func (s *server) fetchPackages(c *gin.Context) {
	token, session, err := getSessionData(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Failed to authenticate")
		return
	}

	apiClient := api.NewClient(s.logger)
	packages, err := apiClient.FetchPackages(token, session)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch packages")
		return
	}
	c.JSON(http.StatusOK, packages)
}

func getSessionData(c *gin.Context) (string, string, error) {
	token, err := c.Cookie("kharma_token")
	if err != nil {
		return "", "", err
	}
	session, err := c.Cookie("kharma_session")
	if err != nil {
		return "", "", err
	}
	return token, session, nil
}
