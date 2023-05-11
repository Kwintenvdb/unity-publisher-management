package server

import (
	"errors"
	"net/http"
	// "time"

	"github.com/Kwintenvdb/unity-publisher-management/api"
	"github.com/Kwintenvdb/unity-publisher-management/logger"

	// jwt "github.com/appleboy/gin-jwt/v2"
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

	r.POST("/authenticate", func(c *gin.Context) {
		email, publisher, err := server.authenticate(c)
		if err != nil {
			c.String(http.StatusUnauthorized, err.Error())
			return
		}
		u := user{
			Email:       email,
			PublisherId: publisher,
		}
		c.JSON(http.StatusOK, u)
	})

	api := r.Group("/api")
	api.GET("/sales/:publisher/:month", server.fetchSales)
	api.GET("/months/:publisher", server.fetchMonths)
	api.GET("/packages", server.fetchPackages)

	r.Run(":8081")
}

func (s *server) authenticate(c *gin.Context) (string, string, error) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	if len(email) == 0 || len(password) == 0 {
		return "", "", errors.New("missing email or password")
	}

	apiClient := api.NewClient(s.logger)
	authResponse, err := apiClient.Authenticate(email, password)
	if err != nil {
		return "", "", errors.New("failed to authenticate")
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

	apiClient := api.NewClient(s.logger)
	sales, err := apiClient.FetchSales(publisher, month, token, session)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch sales")
		return
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
