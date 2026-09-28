// Package secrets selects a concrete SecretsStore implementation at startup
// from the AIGW_SECRETS_PROVIDER setting, keeping the provider choice out of
// the business layer.
package secrets

import (
	"fmt"

	"ai-governance-saas/backend/internal/infrastructure/secrets/envstore"
	"ai-governance-saas/backend/internal/infrastructure/secrets/vaultstore"
	ports "ai-governance-saas/backend/internal/ports/secrets"
)

// Options carries the settings of every supported store; only the fields of
// the selected provider are read.
type Options struct {
	Provider string // env | vault | aws
	// env store:
	File string // encrypted local store path
	Key  string // hex 32-byte AES key; empty = generate <File>.key
	// vault store:
	URL       string // Vault address
	Token     string // X-Vault-Token
	MountPath string // KV v2 mount, default "secret"
}

// New returns the store for the requested provider. The aws provider is
// intentionally not implemented: the business layer is cloud-neutral, and
// adding it later is a drop-in (factory-only) change. Selecting it here is a
// startup error, so a misconfigured production bootstrap fails loudly.
func New(opts Options) (ports.Store, error) {
	switch opts.Provider {
	case "env":
		return envstore.New(opts.File, opts.Key)
	case "vault":
		return vaultstore.New(opts.URL, opts.Token, opts.MountPath)
	case "aws":
		return nil, fmt.Errorf("secrets: provider %q is not implemented; use env (local) or vault (production)", opts.Provider)
	default:
		return nil, fmt.Errorf("secrets: unknown provider %q", opts.Provider)
	}
}
