package auth

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/sukujgrg/go-secretstore"
	"github.com/sukujgrg/ms2pdf/internal/config"
)

var msalCacheKey = secretstore.Key{
	Service: "ms2pdf",
	Account: "msal-cache",
}

type secretCache struct {
	store secretstore.Store
	mu    sync.Mutex
}

func openSecretCache(ctx context.Context) (*secretCache, error) {
	store, err := secretstore.Open(ctx, secretstore.WithInteraction(secretstore.InteractionAllowed))
	if err != nil {
		return nil, err
	}
	return &secretCache{store: store}, nil
}

func (c *secretCache) Replace(ctx context.Context, cache cache.Unmarshaler, _ cache.ReplaceHints) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := c.get(ctx)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	defer clear(data)
	return cache.Unmarshal(data)
}

func (c *secretCache) Export(ctx context.Context, cache cache.Marshaler, _ cache.ExportHints) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := cache.Marshal()
	if err != nil {
		return err
	}
	defer clear(data)
	if len(data) == 0 {
		if err := c.delete(ctx); err != nil {
			return err
		}
		return removeLegacyCacheFile()
	}
	if err := c.store.Set(ctx, msalCacheKey, data); err != nil {
		return err
	}
	return removeLegacyCacheFile()
}

func (c *secretCache) get(ctx context.Context) ([]byte, error) {
	secret, err := c.store.Get(ctx, msalCacheKey)
	if err == nil {
		defer secret.Close()
		return secret.Bytes()
	}
	if secretstore.CodeOf(err) != secretstore.NotFound {
		return nil, err
	}
	data, err := readLegacyCacheFile()
	if err != nil || len(data) == 0 {
		return data, err
	}
	if err := c.store.Set(ctx, msalCacheKey, data); err != nil {
		return data, nil
	}
	_ = removeLegacyCacheFile()
	return data, nil
}

func (c *secretCache) delete(ctx context.Context) error {
	err := c.store.Delete(ctx, msalCacheKey)
	if err != nil && secretstore.CodeOf(err) != secretstore.NotFound {
		return err
	}
	return nil
}

func readLegacyCacheFile() ([]byte, error) {
	p := legacyCachePath()
	if p == "" {
		return nil, nil
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func legacyCachePath() string {
	p, err := config.CachePath()
	if err != nil {
		return ""
	}
	return p
}

func removeLegacyCacheFile() error {
	p := legacyCachePath()
	if p == "" {
		return nil
	}
	err := os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
