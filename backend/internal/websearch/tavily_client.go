package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTavilyTimeout = 15 * time.Second

type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type Client interface {
	Search(ctx context.Context, query string, maxResults int) ([]Result, error)
}

type TavilyClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewTavilyClient(baseURL, apiKey string) *TavilyClient {
	return &TavilyClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: defaultTavilyTimeout,
		},
	}
}

func (c *TavilyClient) Search(ctx context.Context, query string, maxResults int) ([]Result, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, fmt.Errorf("tavily api key is empty")
	}
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return nil, fmt.Errorf("search query is empty")
	}
	if maxResults <= 0 {
		maxResults = 5
	}

	payload := map[string]any{
		"api_key":             c.apiKey,
		"query":               trimmedQuery,
		"search_depth":        "basic",
		"max_results":         maxResults,
		"include_raw_content": false,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal tavily payload: %w", err)
	}

	endpoint := c.baseURL + "/search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create tavily request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call tavily: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tavily response: %w", err)
	}

	var decoded struct {
		Error   string `json:"error"`
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rawBody, &decoded); err != nil {
		return nil, fmt.Errorf("decode tavily response: %w", err)
	}
	if resp.StatusCode >= 400 {
		msg := "tavily request failed"
		if decoded.Error != "" {
			msg = decoded.Error
		}
		return nil, fmt.Errorf("%s (status %d)", msg, resp.StatusCode)
	}

	results := make([]Result, 0, len(decoded.Results))
	for _, item := range decoded.Results {
		results = append(results, Result{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.URL),
			Content: strings.TrimSpace(item.Content),
		})
	}
	return results, nil
}
