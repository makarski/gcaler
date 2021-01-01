package auth

import (
	"context"
	"encoding/json"
	"os"

	"golang.org/x/oauth2"
)

type (
	// GToken is a wrapper for retrieving and caching
	// google access token
	GToken struct {
		oauthCfg  *oauth2.Config
		cacheFile string
		cacheDir  string
	}

	// ConsentHandlerFunc describes a handler callback
	// for exchanging auth. code against access token
	ConsentHandlerFunc func(authURL string) (string, error)
)

// NewGToken inits a GToken struct
func NewGToken(oauthCfg *oauth2.Config, cacheFile, cacheDir string) GToken {
	return GToken{oauthCfg, cacheFile, cacheDir}
}

// Get returns a google access token
func (t GToken) Get(ctx context.Context, handleAuthFunc ConsentHandlerFunc) (*oauth2.Token, error) {
	tkn, err := t.fromCache()
	if err == nil {
		return tkn, nil
	}

	authURL := t.oauthCfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	code, err := handleAuthFunc(authURL)
	if err != nil {
		return nil, err
	}

	tkn, err = t.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	if err := t.cache(tkn); err != nil {
		return nil, err
	}

	return tkn, nil

}

func (t GToken) fromCache() (*oauth2.Token, error) {
	f, err := os.Open(t.cacheFile)
	if err != nil {
		return nil, err
	}

	token := oauth2.Token{}
	if err = json.NewDecoder(f).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (t GToken) cache(tkn *oauth2.Token) error {
	if err := os.MkdirAll(t.cacheDir, os.ModePerm); err != nil {
		return err
	}

	cacheFile, err := os.Create(t.cacheFile)
	if err != nil {
		return err
	}

	return json.NewEncoder(cacheFile).Encode(tkn)
}
