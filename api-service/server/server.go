package server

import (
	"errors"
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
	token, _ := c.Cookie("kharma_token")
	session, _ := c.Cookie("kharma_session")

	s.logger.Debugw("Cookies", "token", token, "session", session)

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
