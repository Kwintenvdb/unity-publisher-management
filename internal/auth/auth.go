package auth

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kwintenvdb/unity-publisher-management/logger"
	"github.com/PuerkitoBio/goquery"
)

// TODO make local variables
const LOGIN_URL = "https://id.unity.com/en/login"
const UNITY_SALES_URL = "https://publisher.assetstore.unity3d.com/sales.html"

func Authenticate(email, password string, client *http.Client, logger logger.Logger) error {
	// Phase 1: Retrieve authenticity token from the login page.
	logger.Debug("Retrieving authenticity token...")

	res, err := client.Get(LOGIN_URL)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}

	form := doc.Find("#new_conversations_create_session_form").First()
	action, exists := form.Attr("action")
	if !exists {
		return errors.New("could not find action attribute on form")
	}

	authenticityToken, exists := form.Find("input[name=\"authenticity_token\"]").First().Attr("value")
	if !exists {
		return errors.New("could not find authenticity token")
	}

	// Phase 2: Log in using retrieved authenticity token and form data.
	logger.Debugw("Logging in...", "authenticity_token", authenticityToken)

	formData := url.Values{
		"utf8":               {"✓"},
		"_method":            {"put"},
		"authenticity_token": {authenticityToken},
		"conversations_create_session_form[email]":    {email},
		"conversations_create_session_form[password]": {password},
		"commit": {"Sign in"},
	}
	loginRes, err := client.PostForm("https://id.unity.com"+action, formData)
	if err != nil {
		return err
	}
	if loginRes.StatusCode != 200 {
		return fmt.Errorf("login failed with status code %d", loginRes.StatusCode)
	}

	// Phase 3: Retrieving session token.
	// We are not yet authenticated for the sales page. It will redirect us to a page from which we need to follow yet another redirect.
	// This redirect URL is embedded in a <meta http-equiv="refresh"> element.
	// Following this URL will retrieve the kharma_session and kharma_token which are used to authenticate against the publisher API.
	// These tokens will be stored in the cookie jar for the upcoming API calls.
	return retrieveSessionCookies(client, logger)
}

func retrieveSessionCookies(client *http.Client, logger logger.Logger) error {
	res, err := client.Get(UNITY_SALES_URL)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return err
	}
	content, exists := doc.Find("meta[http-equiv=\"refresh\"]").First().Attr("content")
	if !exists {
		return errors.New("failed to log in")
	}
	logger.Debug("Logged in successfully. Retrieving session token...")
	split := strings.Split(content, "url=")
	url := split[len(split)-1]
	_, err = client.Get(url)
	return err
}
