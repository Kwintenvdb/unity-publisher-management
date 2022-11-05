package server

import (
	"net/http"

	"github.com/Kwintenvdb/unity-publisher-management/api"
	"github.com/Kwintenvdb/unity-publisher-management/logger"
	"github.com/gofiber/fiber/v2"
)

type server struct {
	apiClient *api.Client
}

func Start() {
	logger := logger.NewLogger()
	server := server{
		apiClient: api.NewClient(logger),
	}

	app := fiber.New()
	app.Post("/authenticate", server.authenticate)
	app.Get("/sales", server.fetchSales)

	logger.Info("Starting server on port 8080")
	app.Listen(":8080")
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

	return c.SendString("Authenticated successfully")
}

func (s *server) fetchSales(c *fiber.Ctx) error {
	sales, err := s.apiClient.FetchSales("202210")
	if err != nil {
		return err
	}
	return c.JSON(sales)
}
