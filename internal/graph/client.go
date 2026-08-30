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
	HTTP     *http.Client
	BaseURL  string
	Token    string
	Progress io.Writer
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

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e == nil {
		return "graph error"
	}
	if e.Code != "" && e.Message != "" {
		return fmt.Sprintf("graph %s: %s (%s)", http.StatusText(e.Status), e.Message, e.Code)
	}
	if e.Message != "" {
		return fmt.Sprintf("graph %s: %s", http.StatusText(e.Status), e.Message)
	}
	return fmt.Sprintf("graph %s", http.StatusText(e.Status))
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
	api := &APIError{Status: resp.StatusCode}
	var ge graphError
	if json.Unmarshal(body, &ge) == nil && (ge.Error.Message != "" || ge.Error.Code != "") {
		api.Code = ge.Error.Code
		if ge.Error.InnerError.Code != "" {
			api.Code = ge.Error.Code + "/" + ge.Error.InnerError.Code
		}
		api.Message = ge.Error.Message
		return api
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	api.Message = msg
	return api
}
