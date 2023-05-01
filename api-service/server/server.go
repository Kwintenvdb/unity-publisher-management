package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
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
	publisher, err := apiClient.Authenticate(email, password)
	if err != nil {
		c.String(http.StatusUnauthorized, "Failed to authenticate")
		return "", "", err
	}

	cookies := apiClient.Cookies()
	for _, cookie := range cookies {
		s.logger.Debugw("Cookie", "name", cookie.Name, "value", cookie.Value)
	}

	c.SetCookie("kharma_token", cookies[1].Value, 0, "", "", false, true)
	c.SetCookie("kharma_session", cookies[2].Value, 0, "", "", false, true)
	return email, publisher, nil
}

type loggingTransport struct{}

func (s *loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	bytes, _ := httputil.DumpRequestOut(r, true)

	resp, err := http.DefaultTransport.RoundTrip(r)
	// err is returned after dumping the response

	respBytes, _ := httputil.DumpResponse(resp, true)
	bytes = append(bytes, respBytes...)

	fmt.Printf("%s\n", bytes)

	return resp, err
}

func (s *server) fetchSales(c *gin.Context) {
	token, _ := c.Cookie("kharma_token")
	session, _ := c.Cookie("kharma_session")

	s.logger.Debugw("Cookies", "token", token, "session", session)

	publisher := c.Param("publisher")
	month := c.Param("month")

	client := http.Client{
		Transport: &loggingTransport{},
	}

	apiClient := api.NewClient(s.logger)
	sales, err := apiClient.FetchSales(&client, publisher, month, token, session)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch sales")
		return
	}
	c.JSON(http.StatusOK, sales)
}
