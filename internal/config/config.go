package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const dirName = "ms2pdf"

type File struct {
	ClientID string `json:"client_id,omitempty"`
	TenantID string `json:"tenant_id,omitempty"`
}

func dirPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, dirName), nil
}

func Dir() (string, error) {
	dir, err := dirPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func CachePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "msal.json"), nil
}

func Load() (File, error) {
	dir, err := dirPath()
	if err != nil {
		return File{}, err
	}
	p := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return File{}, nil
	}
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, err
	}
	return f, nil
}

func Save(f File) error {
	p, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func ResolveClientID(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if v := os.Getenv("MS2PDF_CLIENT_ID"); v != "" {
		return v, nil
	}
	f, err := Load()
	if err != nil {
		return "", err
	}
	if f.ClientID == "" {
		return "", errors.New("no Entra app id: pass --app-id, set MS2PDF_CLIENT_ID, or run login once")
	}
	return f.ClientID, nil
}

func ResolveTenant(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if v := os.Getenv("MS2PDF_TENANT_ID"); v != "" {
		return v, nil
	}
	f, err := Load()
	if err != nil {
		return "", err
	}
	if f.TenantID == "" {
		return "common", nil
	}
	return f.TenantID, nil
}
