package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"

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

func (c *Client) Cookies() []*http.Cookie {
	return c.httpClient.Jar.Cookies(&url.URL{Scheme: "https", Host: "assetstore.unity3d.com"})
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

func (c *Client) FetchSales(client *http.Client, publisher string, month string, token, session string) ([]model.SalesData, error) {
	c.logger.Debugw("Fetching sales...", "month", month)
	
	salesUrl, err := c.getPublisherInfoUrl(publisher, "sales")
	if err != nil {
		return nil, err
	}

	var rawSales model.RawSalesData
	err = c.getJson(client, fmt.Sprintf("%s/%s.json", salesUrl, month), &rawSales, token, session)
	if err != nil {
		c.logger.Errorw("Failed to fetch sales", "error", err, "month", month)
		return nil, err
	}

	return model.SalesFromRaw(rawSales), nil
}

func (c *Client) getPublisherInfoUrl(publisher string, infoType string) (string, error) {
	if publisher == "" {
		return "", errors.New("publisher id is not set")
	}

	const baseUrl = "https://publisher.assetstore.unity3d.com/api/publisher-info"
	return fmt.Sprintf("%s/%s/%s", baseUrl, infoType, publisher), nil
}

func (c *Client) getJson(client *http.Client, url string, v interface{}, token, session string) error {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("x-kharma-token", token)
	req.AddCookie(&http.Cookie{Name: "kharma_session", Value: session})
	req.AddCookie(&http.Cookie{Name: "kharma_token", Value: token})

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
