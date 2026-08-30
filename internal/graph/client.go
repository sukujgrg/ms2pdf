package graph

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultBase = "https://graph.microsoft.com/v1.0"

type Client struct {
	HTTP    *http.Client
	BaseURL string
	Token   string
}

func New(token string) *Client {
	return &Client{
		HTTP: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				// Graph convert returns 302 to a short-lived anonymous download URL.
				req.Header.Del("Authorization")
				return nil
			},
		},
		BaseURL: defaultBase,
		Token:   token,
	}
}

func (c *Client) doGraph(req *http.Request) (*http.Response, error) {
	if c.Token != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, readAPIError(resp)
	}
	return resp, nil
}

type graphError struct {
	Error struct {
		Code       string `json:"code"`
		Message    string `json:"message"`
		InnerError struct {
			Code string `json:"code"`
		} `json:"innerError"`
	} `json:"error"`
}

func readAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	var ge graphError
	if json.Unmarshal(body, &ge) == nil && ge.Error.Message != "" {
		inner := ge.Error.InnerError.Code
		if inner != "" {
			return fmt.Errorf("graph %s: %s (%s/%s)", resp.Status, ge.Error.Message, ge.Error.Code, inner)
		}
		return fmt.Errorf("graph %s: %s (%s)", resp.Status, ge.Error.Message, ge.Error.Code)
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	return fmt.Errorf("graph %s: %s", resp.Status, msg)
}
