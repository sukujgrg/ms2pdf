package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// fragmentSize is a multiple of 320 KiB, as required by Graph upload sessions.
const (
	fragmentSize    = 320 * 1024 * 16
	simpleUploadMax = 4 << 20
	cleanupTimeout  = 30 * time.Second
)

// maxSourceBytes is Graph's commonly cited convert limit. Tests may lower it.
var maxSourceBytes int64 = 100 << 20

type uploadSession struct {
	UploadURL string `json:"uploadUrl"`
}

type driveItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ConvertFile(ctx context.Context, input, output, ext string) (err error) {
	f, err := os.Open(input)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if st.Size() > maxSourceBytes {
		return fmt.Errorf("file is %s; maximum for Graph PDF conversion is %s", formatBytes(st.Size()), formatBytes(maxSourceBytes))
	}

	rep := newReporter(c.Progress)
	name := uuid.NewString() + "." + ext
	remote := oneDrivePath(name)
	uploadLabel := "uploading  " + remote
	rep.update(uploadLabel, 0, st.Size(), true)
	item, err := c.upload(ctx, name, f, st.Size(), rep, uploadLabel)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	defer func() {
		rep.line("deleting  " + remote)
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
		defer cancel()
		if delErr := c.deleteItem(cleanupCtx, item.ID); delErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; also failed to delete temp OneDrive item: %v", err, delErr)
			} else {
				err = fmt.Errorf("converted but failed to delete temp OneDrive item: %w", delErr)
			}
		}
	}()

	if dir := filepath.Dir(output); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(output), ".ms2pdf-*.pdf")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := c.downloadPDF(ctx, item.ID, tmp, rep); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("convert: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return replaceFile(tmpName, output)
}

func replaceFile(tmp, dest string) error {
	if err := os.Rename(tmp, dest); err == nil {
		return nil
	}
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, dest)
}

func oneDrivePath(name string) string {
	return "OneDrive:/" + name
}

func (c *Client) upload(ctx context.Context, name string, r io.ReaderAt, size int64, rep *reporter, label string) (driveItem, error) {
	itemPath := "/me/drive/root:/" + url.PathEscape(name)
	if size <= simpleUploadMax {
		body := rep.reader(io.NewSectionReader(r, 0, size), 0, size, label)
		item, err := c.putContent(ctx, itemPath+":/content", body, size)
		if err == nil {
			rep.update(label, size, size, true)
			rep.finish()
		}
		return item, err
	}
	return c.uploadSession(ctx, itemPath+":/createUploadSession", r, size, rep, label)
}

func (c *Client) putContent(ctx context.Context, path string, body io.Reader, size int64) (driveItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.BaseURL+path, body)
	if err != nil {
		return driveItem{}, err
	}
	req.ContentLength = size
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

func (c *Client) uploadSession(ctx context.Context, sessionPath string, r io.ReaderAt, size int64, rep *reporter, label string) (driveItem, error) {
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
	return c.putFragments(ctx, session.UploadURL, r, size, rep, label)
}

func (c *Client) putFragments(ctx context.Context, uploadURL string, r io.ReaderAt, total int64, rep *reporter, label string) (driveItem, error) {
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
	for start := int64(0); start < total; start += fragmentSize {
		end := start + fragmentSize
		if end > total {
			end = total
		}
		chunk := rep.reader(io.NewSectionReader(r, start, end-start), start, total, label)
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, chunk)
		if err != nil {
			return driveItem{}, err
		}
		req.ContentLength = end - start
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
		rep.update(label, end, total, true)
	}
	if last.ID == "" {
		return driveItem{}, fmt.Errorf("upload did not return an item id")
	}
	rep.finish()
	return last, nil
}

func (c *Client) downloadPDF(ctx context.Context, itemID string, w io.Writer, rep *reporter) error {
	rep.line("converting…")
	u := c.BaseURL + "/me/drive/items/" + itemID + "/content?format=pdf"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.doGraph(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(rep.writer(w, maxSourceBytes, "downloading"), resp.Body)
	if err == nil {
		rep.finish()
	}
	return err
}

func (c *Client) deleteItem(ctx context.Context, itemID string) error {
	if err := c.permanentDelete(ctx, itemID); err == nil {
		return nil
	} else if !isPermanentDeleteUnsupported(err) {
		return err
	}
	if err := c.recycleDelete(ctx, itemID); err != nil {
		return err
	}
	if err := c.permanentDelete(ctx, itemID); err == nil || isNotFound(err) {
		return nil
	} else if !isPermanentDeleteUnsupported(err) {
		return err
	}
	if err := c.recycleDelete(ctx, itemID); err != nil && !isNotFound(err) {
		return fmt.Errorf("temp item remains in OneDrive recycle bin: %w", err)
	}
	return nil
}

func (c *Client) permanentDelete(ctx context.Context, itemID string) error {
	u := c.BaseURL + "/me/drive/items/" + itemID + "/permanentDelete"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, http.NoBody)
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

func (c *Client) recycleDelete(ctx context.Context, itemID string) error {
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

func isPermanentDeleteUnsupported(err error) bool {
	var api *APIError
	if !errors.As(err, &api) {
		return false
	}
	if api.Status == http.StatusNotFound || api.Status == http.StatusMethodNotAllowed || api.Status == http.StatusNotImplemented {
		return true
	}
	msg := strings.ToLower(api.Code + " " + api.Message)
	return strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "api not found") ||
		strings.Contains(msg, "unknownerror")
}

func isNotFound(err error) bool {
	var api *APIError
	if !errors.As(err, &api) {
		return false
	}
	return api.Status == http.StatusNotFound || strings.EqualFold(api.Code, "itemNotFound")
}
