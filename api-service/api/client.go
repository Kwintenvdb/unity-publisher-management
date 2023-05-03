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
}

func NewClient(logger logger.Logger) *Client {
	return &Client{
		logger:     logger,
	}
}

type authenticationResponse struct {
	PublisherId   string
	KharmaToken   string
	KharmaSession string
}

// Authenticate and cache the publisher id
func (c *Client) Authenticate(email, password string) (*authenticationResponse, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Jar: jar,
	}

	err = auth.Authenticate(email, password, client, c.logger)
	if err != nil {
		c.logger.Errorw("Failed to authenticate", "error", err)
		return nil, err
	}

	id, err := c.fetchPublisherId(client)
	if err != nil {
		return nil, err
	}

	token, session, err := extractKharmaCookies(jar.Cookies(&url.URL{Scheme: "https", Host: "assetstore.unity3d.com"}))
	if err != nil {
		return nil, err
	}

	return &authenticationResponse{
		PublisherId:   id,
		KharmaToken:   token,
		KharmaSession: session,
	}, nil
}

func extractKharmaCookies(cookies []*http.Cookie) (string, string, error) {
	var token, session string
	for _, cookie := range cookies {
		if cookie.Name == "kharma_token" {
			token = cookie.Value
		}
		if cookie.Name == "kharma_session" {
			session = cookie.Value
		}
	}
	if token == "" || session == "" {
		return "", "", errors.New("could not find kharma_token or kharma_session cookie")
	}
	return token, session, nil
}

func (c *Client) fetchPublisherId(client *http.Client) (string, error) {
	c.logger.Debug("Fetching publisher id...")
	overview, err := c.fetchOverview(client)
	return overview.Id, err
}

func (c *Client) fetchOverview(client *http.Client) (model.Overview, error) {
	c.logger.Debug("Fetching overview...")

	// Fetch the overview data
	res, err := client.Get("https://publisher.assetstore.unity3d.com/api/publisher/overview.json")
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

func (c *Client) FetchSales(publisher, month, token, session string) ([]model.SalesData, error) {
	c.logger.Debugw("Fetching sales...", "month", month)

	salesUrl, err := c.getPublisherInfoUrl(publisher, "sales")
	if err != nil {
		return nil, err
	}

	var rawSales model.RawSalesData
	err = c.getJson(fmt.Sprintf("%s/%s.json", salesUrl, month), &rawSales, token, session)
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

func (c *Client) getJson(url string, v interface{}, token, session string) error {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("x-kharma-token", token)
	req.AddCookie(&http.Cookie{Name: "kharma_session", Value: session})
	req.AddCookie(&http.Cookie{Name: "kharma_token", Value: token})

	client := http.Client{}
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
