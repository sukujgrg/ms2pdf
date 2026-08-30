package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// fragmentSize is a multiple of 320 KiB, as required by Graph upload sessions.
const (
	fragmentSize    = 320 * 1024 * 16
	simpleUploadMax = 4 << 20
)

type uploadSession struct {
	UploadURL string `json:"uploadUrl"`
}

type driveItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ConvertFile(ctx context.Context, input, output, ext string) error {
	data, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	name := uuid.NewString() + "." + ext
	item, err := c.upload(ctx, name, data)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	defer func() { _ = c.deleteItem(ctx, item.ID) }()

	pdf, err := c.downloadPDF(ctx, item.ID)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	if dir := filepath.Dir(output); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(output, pdf, 0o644)
}

func (c *Client) upload(ctx context.Context, name string, data []byte) (driveItem, error) {
	itemPath := "/me/drive/root:/" + url.PathEscape(name)
	if len(data) <= simpleUploadMax {
		return c.putContent(ctx, itemPath+":/content", data)
	}
	return c.uploadSession(ctx, itemPath+":/createUploadSession", data)
}

func (c *Client) putContent(ctx context.Context, path string, data []byte) (driveItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return driveItem{}, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := c.doGraph(req)
	if err != nil {
		return driveItem{}, err
	}
	defer resp.Body.Close()
	var item driveItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return driveItem{}, err
	}
	if item.ID == "" {
		return driveItem{}, fmt.Errorf("upload did not return an item id")
	}
	return item, nil
}

func (c *Client) uploadSession(ctx context.Context, sessionPath string, data []byte) (driveItem, error) {
	body, _ := json.Marshal(map[string]any{
		"item": map[string]string{
			"@microsoft.graph.conflictBehavior": "replace",
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+sessionPath, bytes.NewReader(body))
	if err != nil {
		return driveItem{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.doGraph(req)
	if err != nil {
		return driveItem{}, err
	}
	var session uploadSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		resp.Body.Close()
		return driveItem{}, err
	}
	resp.Body.Close()
	if session.UploadURL == "" {
		return driveItem{}, fmt.Errorf("empty uploadUrl")
	}
	return c.putFragments(ctx, session.UploadURL, data)
}

func (c *Client) putFragments(ctx context.Context, uploadURL string, data []byte) (driveItem, error) {
	total := len(data)
	if total == 0 {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, http.NoBody)
		if err != nil {
			return driveItem{}, err
		}
		req.Header.Set("Content-Range", "bytes */0")
		resp, err := c.do(req)
		if err != nil {
			return driveItem{}, err
		}
		var last driveItem
		if err := json.NewDecoder(resp.Body).Decode(&last); err != nil {
			resp.Body.Close()
			return driveItem{}, err
		}
		resp.Body.Close()
		if last.ID == "" {
			return driveItem{}, fmt.Errorf("upload did not return an item id")
		}
		return last, nil
	}
	var last driveItem
	for start := 0; start < total; start += fragmentSize {
		end := start + fragmentSize
		if end > total {
			end = total
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(data[start:end]))
		if err != nil {
			return driveItem{}, err
		}
		req.Header.Set("Content-Length", fmt.Sprintf("%d", end-start))
		req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, total))
		resp, err := c.do(req)
		if err != nil {
			return driveItem{}, err
		}
		if end == total {
			if err := json.NewDecoder(resp.Body).Decode(&last); err != nil {
				resp.Body.Close()
				return driveItem{}, err
			}
		}
		resp.Body.Close()
	}
	if last.ID == "" {
		return driveItem{}, fmt.Errorf("upload did not return an item id")
	}
	return last, nil
}

func (c *Client) downloadPDF(ctx context.Context, itemID string) ([]byte, error) {
	u := c.BaseURL + "/me/drive/items/" + itemID + "/content?format=pdf"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.doGraph(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) deleteItem(ctx context.Context, itemID string) error {
	u := c.BaseURL + "/me/drive/items/" + itemID
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.doGraph(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
