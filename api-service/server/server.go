package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/Kwintenvdb/unity-publisher-management/api"
	"github.com/Kwintenvdb/unity-publisher-management/logger"

	"github.com/gin-gonic/gin"
)

type server struct {
	apiClient *api.Client
	logger    logger.Logger
}

func Start() {
	logger := logger.NewLogger()
	server := server{
		apiClient: api.NewClient(logger),
		logger:    logger,
	}

	r := gin.Default()

	r.POST("/authenticate", server.authenticate)
	r.GET("/sales/:publisher/:month", server.fetchSales)

	logger.Info("Starting server on port 8081")
	r.Run(":8081")
}

func (s *server) authenticate(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	if len(email) == 0 || len(password) == 0 {
		c.String(http.StatusBadRequest, "Missing email or password")
		return
	}

	if err := s.apiClient.Authenticate(email, password); err != nil {
		c.String(http.StatusUnauthorized, "Failed to authenticate")
		return
	}

	cookies := s.apiClient.Cookies()
	for _, cookie := range cookies {
		s.logger.Debugw("Cookie", "name", cookie.Name, "value", cookie.Value)
	}

	c.SetCookie("kharma_token", cookies[1].Value, 0, "", "", false, true)
	c.SetCookie("kharma_session", cookies[2].Value, 0, "", "", false, true)
	c.String(http.StatusOK, "Authenticated successfully")
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

	sales, err := s.apiClient.FetchSales(&client, publisher, month, token, session)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch sales")
		return
	}
	c.JSON(http.StatusOK, sales)
}
