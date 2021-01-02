package auth

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

type (
	// GToken is a wrapper for retrieving and caching
	// google access token
	GToken struct {
		credentialsFile string
		cacheFile       string
		cacheDir        string

		oauthCfg *oauth2.Config
	}

	// ConsentHandlerFunc describes a handler callback
	// for exchanging auth. code against access token
	ConsentHandlerFunc func(authURL string) (string, error)
)

// NewGToken lazy inits a GToken struct
func NewGToken(credentialsFile, cacheFile, cacheDir string) GToken {
	return GToken{credentialsFile, cacheFile, cacheDir, nil}
}

// Get returns a google access token
func (t GToken) Get(ctx context.Context, handleAuthFunc ConsentHandlerFunc) (*oauth2.Token, error) {
	tkn, err := t.fromCache()
	if err == nil {
		return tkn, nil
	}

	oauthCfg, err := t.Credentials()
	if err != nil {
		return nil, err
	}

	authURL := oauthCfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	code, err := handleAuthFunc(authURL)
	if err != nil {
		return nil, err
	}

	tkn, err = oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	if err := t.cache(tkn); err != nil {
		return nil, err
	}

	return tkn, nil

}

// Credentials reads google client config file
// and returns a deserialized struct on success
func (t *GToken) Credentials() (*oauth2.Config, error) {
	if t.oauthCfg != nil {
		return t.oauthCfg, nil
	}

	b, err := ioutil.ReadFile(t.credentialsFile)
	if err != nil {
		return nil, err
	}

	cfg, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		return nil, err
	}

	t.oauthCfg = cfg

	return cfg, nil
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
