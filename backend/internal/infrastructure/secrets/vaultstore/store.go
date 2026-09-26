// Package vaultstore implements the SecretsStore port against HashiCorp
// Vault KV v2, using only the standard library HTTP client. It is the
// production store: secrets live in Vault, never in the application
// process, and TLS terminates at Vault.
package vaultstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	ports "ai-governance-saas/backend/internal/ports/secrets"
)

// Store is a minimal Vault KV v2 REST client. Each secret reference is a
// KV v2 path; values are stored under the single data field "value", so the
// reference and the stored shape stay flat.
type Store struct {
	baseURL   *url.URL
	token     string
	client    *http.Client
	mountPath string
}

// New returns a Store for the Vault at baseAddr (e.g. https://vault.
// example.com:8200). The mount path defaults to "secret" (KV v2 mounts are
// served under /v1/<mount>/data/...); token must be the X-Vault-Token.
func New(baseAddr, token, mountPath string) (*Store, error) {
	u, err := url.Parse(baseAddr)
	if err != nil {
		return nil, fmt.Errorf("vaultstore: parse base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("vaultstore: base url must include scheme and host")
	}
	if token == "" {
		return nil, fmt.Errorf("vaultstore: token must not be empty")
	}
	if mountPath == "" {
		mountPath = "secret"
	}
	return &Store{
		baseURL:   u,
		token:     token,
		mountPath: mountPath,
		client:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Get returns the stored value for ref, or ErrNotFound.
func (s *Store) Get(ctx context.Context, ref string) (string, error) {
	var out struct {
		Data struct {
			Data struct {
				Value string `json:"value"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := s.do(ctx, http.MethodGet, dataPath(s.baseURL, s.mountPath, ref), nil, &out); err != nil {
		if isNotFound(err) {
			return "", ports.ErrNotFound
		}
		return "", err
	}
	return out.Data.Data.Value, nil
}

// Put writes value under ref.
func (s *Store) Put(ctx context.Context, ref, value string) error {
	body := struct {
		Data map[string]string `json:"data"`
	}{Data: map[string]string{"value": value}}
	return s.do(ctx, http.MethodPut, dataPath(s.baseURL, s.mountPath, ref), body, nil)
}

// Delete removes ref (KV v2 delete is a tombstone).
func (s *Store) Delete(ctx context.Context, ref string) error {
	return s.do(ctx, http.MethodDelete, dataPath(s.baseURL, s.mountPath, ref), nil, nil)
}

func dataPath(base *url.URL, mount, ref string) string {
	u := *base
	u.Path = joinURLPath(u.Path, "/v1/", mount, "/data/", ref)
	return u.String()
}

func joinURLPath(parts ...string) string {
	out := ""
	for _, p := range parts {
		out += p
	}
	return out
}

// do performs one authenticated Vault request. A JSON response body is
// decoded into out when out is non-nil; 404 maps to a sentinel that Get
// translates to ErrNotFound.
func (s *Store) do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("vaultstore: encode request: %w", err)
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, path, rdr)
	if err != nil {
		return fmt.Errorf("vaultstore: build request: %w", err)
	}
	req.Header.Set("X-Vault-Token", s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("vaultstore: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errVaultNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("vaultstore: %s %s: status %s: %s", method, path, resp.Status, string(raw))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("vaultstore: decode response: %w", err)
		}
	}
	return nil
}

type notFoundError struct{}

func (notFoundError) Error() string { return "vaultstore: not found" }

func isNotFound(err error) bool {
	_, ok := err.(notFoundError)
	return ok
}

var errVaultNotFound = notFoundError{}