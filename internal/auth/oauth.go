package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/voloshmiak/tamagochi/internal/platform/apperr"
)

const (
	ProviderGoogle = "google"
	ProviderGitHub = "github"
)

type Identity struct {
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Login         string
}

type OAuthProvider interface {
	Name() string
	AuthCodeURL(state, verifier string) string
	Identify(ctx context.Context, code, verifier string) (Identity, error)
}

var (
	ErrUnknownProvider = fmt.Errorf("%w: unknown or disabled OAuth provider", apperr.ErrNotFound)
	ErrProviderFailed  = errors.New("oauth provider error")
	ErrNoVerifiedEmail = fmt.Errorf("%w: the provider did not share a verified email", apperr.ErrInvalid)
)

var googleEndpoint = oauth2.Endpoint{
	AuthURL:   "https://accounts.google.com/o/oauth2/auth",
	TokenURL:  "https://oauth2.googleapis.com/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type oauthProvider struct {
	name     string
	conf     *oauth2.Config
	client   *http.Client
	identify func(ctx context.Context, c *http.Client) (Identity, error)
}

func (p *oauthProvider) Name() string { return p.name }

func (p *oauthProvider) AuthCodeURL(state, verifier string) string {
	return p.conf.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
}

func (p *oauthProvider) Identify(ctx context.Context, code, verifier string) (Identity, error) {
	if p.client != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	}
	tok, err := p.conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return Identity{}, fmt.Errorf("%w: exchange code: %v", ErrProviderFailed, err)
	}
	id, err := p.identify(ctx, p.conf.Client(ctx, tok))
	if err != nil {
		return Identity{}, err
	}
	id.Provider = p.name
	return id, nil
}

func NewGoogleProvider(c OAuthConfig) OAuthProvider {
	return &oauthProvider{
		name: ProviderGoogle,
		conf: &oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			RedirectURL:  c.RedirectURL,
			Endpoint:     googleEndpoint,
			Scopes:       []string{"openid", "email", "profile"},
		},
		identify: googleIdentity("https://openidconnect.googleapis.com/v1/userinfo"),
	}
}

func NewGitHubProvider(c OAuthConfig) OAuthProvider {
	return &oauthProvider{
		name: ProviderGitHub,
		conf: &oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			RedirectURL:  c.RedirectURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		},
		identify: githubIdentity("https://api.github.com"),
	}
}

func googleIdentity(userinfoURL string) func(ctx context.Context, c *http.Client) (Identity, error) {
	return func(ctx context.Context, c *http.Client) (Identity, error) {
		var info struct {
			Sub           string `json:"sub"`
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
			Name          string `json:"name"`
		}
		if err := getJSON(ctx, c, userinfoURL, &info); err != nil {
			return Identity{}, err
		}
		if info.Sub == "" {
			return Identity{}, fmt.Errorf("%w: userinfo without subject", ErrProviderFailed)
		}
		return Identity{Subject: info.Sub, Email: info.Email, EmailVerified: info.EmailVerified, Name: info.Name}, nil
	}
}

func githubIdentity(apiURL string) func(ctx context.Context, c *http.Client) (Identity, error) {
	return func(ctx context.Context, c *http.Client) (Identity, error) {
		var user struct {
			ID    int64  `json:"id"`
			Login string `json:"login"`
			Name  string `json:"name"`
		}
		if err := getJSON(ctx, c, apiURL+"/user", &user); err != nil {
			return Identity{}, err
		}
		if user.ID == 0 {
			return Identity{}, fmt.Errorf("%w: user without id", ErrProviderFailed)
		}
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := getJSON(ctx, c, apiURL+"/user/emails", &emails); err != nil {
			return Identity{}, err
		}
		id := Identity{Subject: strconv.FormatInt(user.ID, 10), Name: user.Name, Login: user.Login}
		for _, e := range emails {
			if e.Verified && (e.Primary || id.Email == "") {
				id.Email, id.EmailVerified = e.Email, true
			}
		}
		return id, nil
	}
}

func getJSON(ctx context.Context, c *http.Client, url string, v any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s answered %d", ErrProviderFailed, url, resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(v); err != nil {
		return fmt.Errorf("%w: decode %s: %v", ErrProviderFailed, url, err)
	}
	return nil
}
