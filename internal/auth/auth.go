package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

var (
	ErrNotLoggedIn = errors.New("not logged in; run login first")
	scopes         = []string{
		"https://graph.microsoft.com/Files.ReadWrite",
		"https://graph.microsoft.com/User.Read",
	}
)

type Session struct {
	client        public.Client
	tenant        string
	homeAccountID string
}

type Account struct {
	HomeAccountID string
	Username      string
	Name          string
}

func Authority(tenant string) (string, error) {
	t := strings.TrimSpace(tenant)
	t = strings.TrimRight(t, "/")
	if t == "" {
		t = "common"
	}
	if strings.Contains(t, "://") {
		return t, nil
	}
	return "https://login.microsoftonline.com/" + t, nil
}

func DeviceCodeTenantOK(tenant string) bool {
	t := strings.ToLower(strings.TrimSpace(tenant))
	return t != "" && t != "common" && t != "consumers" && t != "organizations" && !strings.Contains(t, "://")
}

func Open(ctx context.Context, clientID, tenant, homeAccountID string) (*Session, error) {
	if strings.TrimSpace(tenant) == "" {
		tenant = "common"
	}
	authority, err := Authority(tenant)
	if err != nil {
		return nil, err
	}
	tokCache, err := openSecretCache(ctx)
	if err != nil {
		return nil, fmt.Errorf("open keychain: %w", err)
	}
	client, err := public.New(clientID,
		public.WithCache(tokCache),
		public.WithAuthority(authority),
	)
	if err != nil {
		return nil, err
	}
	return &Session{client: client, tenant: tenant, homeAccountID: strings.TrimSpace(homeAccountID)}, nil
}

func (s *Session) Login(ctx context.Context, out io.Writer) (Account, error) {
	if _, err := fmt.Fprintln(out, "Opening a browser. Sign in with any Microsoft account (personal or work)."); err != nil {
		return Account{}, err
	}
	result, err := s.client.AcquireTokenInteractive(ctx, scopes, public.WithRedirectURI("http://localhost"))
	if err != nil {
		return Account{}, fmt.Errorf("browser login: %w", wrapLoginErr(err))
	}
	if err := s.keepOnly(ctx, result.Account); err != nil {
		return Account{}, err
	}
	return accountFrom(result.Account), nil
}

func (s *Session) LoginDeviceCode(ctx context.Context, out io.Writer) (Account, error) {
	if !DeviceCodeTenantOK(s.tenant) {
		return Account{}, errors.New("device-code login needs --tenant with a Directory (tenant) ID; it cannot sign in personal Microsoft accounts")
	}
	opts := []public.AcquireByDeviceCodeOption{public.WithTenantID(s.tenantID())}
	dc, err := s.client.AcquireTokenByDeviceCode(ctx, scopes, opts...)
	if err != nil {
		return Account{}, fmt.Errorf("start device code: %w", wrapTenantErr(err))
	}
	if _, err := fmt.Fprintln(out, dc.Result.Message); err != nil {
		return Account{}, err
	}
	result, err := dc.AuthenticationResult(ctx)
	if err != nil {
		return Account{}, fmt.Errorf("complete device code: %w", wrapTenantErr(err))
	}
	if err := s.keepOnly(ctx, result.Account); err != nil {
		return Account{}, err
	}
	return accountFrom(result.Account), nil
}

func (s *Session) SilentToken(ctx context.Context) (string, error) {
	acct, err := s.activeAccount(ctx)
	if err != nil {
		return "", err
	}
	opts := []public.AcquireSilentOption{public.WithSilentAccount(acct)}
	if s.tenantID() != "" {
		opts = append(opts, public.WithTenantID(s.tenantID()))
	}
	result, err := s.client.AcquireTokenSilent(ctx, scopes, opts...)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotLoggedIn, err)
	}
	return result.AccessToken, nil
}

func wrapLoginErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "AADSTS70002") || strings.Contains(msg, "AADSTS7000218") || strings.Contains(msg, "client_secret") {
		return fmt.Errorf("%w\nEntra thinks this is a web app. In Authentication: delete any Web redirect, add platform \"Mobile and desktop applications\" with http://localhost, and set Allow public client flows = Yes. Do not add a client secret", err)
	}
	return wrapTenantErr(err)
}

func wrapTenantErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "AADSTS50059") || strings.Contains(msg, "AADSTS90133") {
		return fmt.Errorf("%w (pass --tenant with the Directory (tenant) ID from Entra Overview)", err)
	}
	return err
}

func (s *Session) tenantID() string {
	if !DeviceCodeTenantOK(s.tenant) {
		return ""
	}
	return s.tenant
}

func (s *Session) WhoAmI(ctx context.Context) (Account, error) {
	acct, err := s.activeAccount(ctx)
	if err != nil {
		return Account{}, err
	}
	return accountFrom(acct), nil
}

func (s *Session) activeAccount(ctx context.Context) (public.Account, error) {
	accounts, err := s.client.Accounts(ctx)
	if err != nil {
		return public.Account{}, err
	}
	return accountByHomeID(accounts, s.homeAccountID)
}

func accountByHomeID(accounts []public.Account, homeID string) (public.Account, error) {
	if homeID == "" || len(accounts) == 0 {
		return public.Account{}, ErrNotLoggedIn
	}
	for _, a := range accounts {
		if a.HomeAccountID == homeID {
			return a, nil
		}
	}
	return public.Account{}, ErrNotLoggedIn
}

func (s *Session) keepOnly(ctx context.Context, keep public.Account) error {
	accounts, err := s.client.Accounts(ctx)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		if a.HomeAccountID == keep.HomeAccountID {
			continue
		}
		if err := s.client.RemoveAccount(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) Logout(ctx context.Context) error {
	accounts, err := s.client.Accounts(ctx)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		if err := s.client.RemoveAccount(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

func accountFrom(a public.Account) Account {
	name := a.Name
	if name == "" {
		name = a.PreferredUsername
	}
	return Account{HomeAccountID: a.HomeAccountID, Username: a.PreferredUsername, Name: name}
}
