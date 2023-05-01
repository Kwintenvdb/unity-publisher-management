package server

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"

	"github.com/Kwintenvdb/unity-publisher-management/api"
	"github.com/Kwintenvdb/unity-publisher-management/logger"
	"github.com/gofiber/fiber/v2"
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

	app := fiber.New()
	app.Post("/authenticate", server.authenticate)
	app.Get("/sales/:publisher/:month", server.fetchSales)

	logger.Info("Starting server on port 8081")
	app.Listen(":8081")
}

func (s *server) authenticate(c *fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	if len(email) == 0 || len(password) == 0 {
		c.SendString("Missing email or password")
		return c.SendStatus(http.StatusBadRequest)
	}

	if err := s.apiClient.Authenticate(email, password); err != nil {
		c.SendString("Failed to authenticate")
		return c.SendStatus(http.StatusUnauthorized)
	}

	cookies := s.apiClient.Cookies()
	for _, cookie := range cookies {
		s.logger.Debugw("Cookie", "name", cookie.Name, "value", cookie.Value)
	}

	c.Cookie(&fiber.Cookie{
		Name:  "kharma_token",
		Value: cookies[1].Value,
	})

	c.Cookie(&fiber.Cookie{
		Name:  "kharma_session",
		Value: cookies[2].Value,
	})

	return c.SendString("Authenticated successfully")
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

func (s *server) fetchSales(c *fiber.Ctx) error {
	token := c.Cookies("kharma_token")
	session := c.Cookies("kharma_session")

	s.logger.Debugw("Cookies", "token", token, "session", session)

	publisher := c.Params("publisher")
	month := c.Params("month")

	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}


	client := http.Client{
		Jar: jar,
		Transport: &loggingTransport{},
	}

	sales, err := s.apiClient.FetchSales(&client, publisher, month, token, session)
	if err != nil {
		return err
	}
	return c.JSON(sales)
}
