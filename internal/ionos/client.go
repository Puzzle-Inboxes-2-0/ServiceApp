package ionos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Client is the IONOS Cloud API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewClient creates a new IONOS API client
func NewClient(baseURL, token string, logger *logrus.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// IPBlockProperties holds the properties of an IP block
type IPBlockProperties struct {
	Name     string   `json:"name"`
	Location string   `json:"location"`
	Size     int      `json:"size"`
	IPs      []string `json:"ips"`
}

// IPBlock represents an IONOS IP block
type IPBlock struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Href       string            `json:"href"`
	Properties IPBlockProperties `json:"properties"`
}

// IPBlockResponse is the response from creating an IP block
type IPBlockResponse struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Href       string            `json:"href"`
	Properties IPBlockProperties `json:"properties"`
}

// IPBlocksResponse is the response when listing IP blocks
type IPBlocksResponse struct {
	ID    string    `json:"id"`
	Type  string    `json:"type"`
	Href  string    `json:"href"`
	Items []IPBlock `json:"items"`
}

// ReserveIPBlockRequest is the request to reserve an IP block
type ReserveIPBlockRequest struct {
	Properties IPBlockProperties `json:"properties"`
}

// ReserveIPBlock reserves a new IP block in IONOS
func (c *Client) ReserveIPBlock(ctx context.Context, location string, size int, name string) (*IPBlockResponse, error) {
	if name == "" {
		name = fmt.Sprintf("Auto-reserved-%d", time.Now().Unix())
	}

	reqBody := ReserveIPBlockRequest{
		Properties: IPBlockProperties{
			Name:     name,
			Location: location,
			Size:     size,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/ipblocks", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(logrus.Fields{
		"action":   "reserve_ip_block",
		"location": location,
		"size":     size,
		"name":     name,
	}).Info("Reserving IP block")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var ipBlock IPBlockResponse
	if err := json.Unmarshal(body, &ipBlock); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"action":   "reserve_ip_block",
		"block_id": ipBlock.ID,
		"ips":      ipBlock.Properties.IPs,
	}).Info("IP block reserved successfully")

	return &ipBlock, nil
}

// GetIPBlock retrieves an IP block by ID
func (c *Client) GetIPBlock(ctx context.Context, blockID string) (*IPBlockResponse, error) {
	url := fmt.Sprintf("%s/ipblocks/%s", c.baseURL, blockID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var ipBlock IPBlockResponse
	if err := json.Unmarshal(body, &ipBlock); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &ipBlock, nil
}

// ListIPBlocks lists all IP blocks
func (c *Client) ListIPBlocks(ctx context.Context) (*IPBlocksResponse, error) {
	url := fmt.Sprintf("%s/ipblocks?depth=2", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var blocks IPBlocksResponse
	if err := json.Unmarshal(body, &blocks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &blocks, nil
}

// DeleteIPBlock deletes an IP block by ID
func (c *Client) DeleteIPBlock(ctx context.Context, blockID string) error {
	url := fmt.Sprintf("%s/ipblocks/%s", c.baseURL, blockID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(logrus.Fields{
		"action":   "delete_ip_block",
		"block_id": blockID,
	}).Info("Deleting IP block")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	c.logger.WithFields(logrus.Fields{
		"action":   "delete_ip_block",
		"block_id": blockID,
	}).Info("IP block deleted successfully")

	return nil
}

