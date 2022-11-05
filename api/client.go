package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"

	"github.com/Kwintenvdb/unity-publisher-management/api/model"
	"github.com/Kwintenvdb/unity-publisher-management/internal/auth"
	"github.com/Kwintenvdb/unity-publisher-management/logger"
)

type Client struct {
	logger     logger.Logger
	httpClient *http.Client
	// publisherId will be set after calling Authenticate
	publisherId string
}

func NewClient(logger logger.Logger) *Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Jar: jar,
	}

	return &Client{
		logger:     logger,
		httpClient: client,
	}
}

// Authenticate and cache the publisher id
func (c *Client) Authenticate(email, password string) error {
	err := auth.Authenticate(email, password, c.httpClient, c.logger)
	if err != nil {
		c.logger.Errorw("Failed to authenticate", "error", err)
		return err
	}
	id, err := c.fetchPublisherId()
	if err != nil {
		return err
	}
	c.logger.Debugw("Fetched publisher id", "publisher_id", id)
	c.publisherId = id
	return nil
}

func (c *Client) fetchPublisherId() (string, error) {
	c.logger.Debug("Fetching publisher id...")
	overview, err := c.fetchOverview()
	return overview.Id, err
}

func (c *Client) fetchOverview() (model.Overview, error) {
	c.logger.Debug("Fetching overview...")

	// Fetch the overview data
	res, err := c.httpClient.Get("https://publisher.assetstore.unity3d.com/api/publisher/overview.json")
	if err != nil {
		return model.Overview{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return model.Overview{}, err
	}

	// Unmarshal json
	var data struct {
		Overview model.Overview `json:"overview"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return model.Overview{}, err
	}
	return data.Overview, nil
}
