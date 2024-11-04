package webmention

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/charmbracelet/log"
)

var BaseUrl = "https://webmention.io/api/mentions.jf2" // API endpoint

type Client struct {
	Domain   string
	Token    string
	SinceID  int
	PageSize int
	page     int
}

type Getter interface {
	GetMentions() (*Response, error)
}

func (client *Client) GetMentions() (*Response, error) {
	// Define query parameters
	params := url.Values{}
	params.Add("domain", client.Domain)
	params.Add("token", client.Token)
	params.Add("per-page", strconv.Itoa(client.PageSize))
	params.Add("since_id", strconv.Itoa(client.SinceID))
	params.Add("page", strconv.Itoa(client.page))

	// Construct the full URL with query parameters
	requestURL := fmt.Sprintf("%s?%s", BaseUrl, params.Encode())

	log.Debug("Querying api", "url", requestURL, "params", params)
	// Send the GET request
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching webmentions: %v", err.Error())
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(fmt.Sprintf("error closing response.Body: %v", err))
		}
	}(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("error fetching webmentions: %s", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}
	client.page++
	return &result, nil
}
