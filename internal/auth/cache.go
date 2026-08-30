package auth

import (
	"context"
	"sync"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/sukujgrg/go-secretstore"
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
		return c.delete(ctx)
	}
	return c.store.Set(ctx, msalCacheKey, data)
}

func (c *secretCache) get(ctx context.Context) ([]byte, error) {
	secret, err := c.store.Get(ctx, msalCacheKey)
	if err == nil {
		defer secret.Close()
		return secret.Bytes()
	}
	if secretstore.CodeOf(err) == secretstore.NotFound {
		return nil, nil
	}
	return nil, err
}

func (c *secretCache) delete(ctx context.Context) error {
	err := c.store.Delete(ctx, msalCacheKey)
	if err != nil && secretstore.CodeOf(err) != secretstore.NotFound {
		return err
	}
	return nil
}
