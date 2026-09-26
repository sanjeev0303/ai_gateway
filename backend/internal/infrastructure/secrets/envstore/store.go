// Package envstore implements the SecretsStore port with an AES-256-GCM
// encrypted JSON file. It is the local-development store: single process,
// file locked in-memory only. Production deployments use the vault store.
package envstore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	ports "ai-governance-saas/backend/internal/ports/secrets"
)

const envelopeVersion = 1

// Item is one encrypted value: a random 12-byte nonce and the GCM
// ciphertext (which carries its own 16-byte authentication tag).
type item struct {
	Nonce  string `json:"nonce"`
	Cipher string `json:"cipher"`
}

// Envelope is the on-disk shape: versioned, one item per reference.
type envelope struct {
	Version int             `json:"version"`
	Secrets map[string]item `json:"secrets"`
}

// Store is a file-backed, AES-256-GCM encrypted secret store. All methods
// are safe for concurrent use within one process.
type Store struct {
	path string
	key  [32]byte
	mu   sync.Mutex
}

// New opens (or creates) the store at path. When keyHex is empty the key is
// read from path+".key"; if that file does not exist a fresh key is generated
// and persisted there with mode 0600, so a store never re-keys itself.
func New(path, keyHex string) (*Store, error) {
	key, err := loadKey(path, keyHex)
	if err != nil {
		return nil, err
	}
	s := &Store{path: path, key: key}
	if err := s.initFile(); err != nil {
		return nil, err
	}
	return s, nil
}

func loadKey(path, keyHex string) ([32]byte, error) {
	var key [32]byte
	if keyHex != "" {
		decoded, err := hex.DecodeString(keyHex)
		if err != nil || len(decoded) != 32 {
			return key, errors.New("envstore: AIGW_SECRETS_KEY must be hex-encoded 32 bytes")
		}
		copy(key[:], decoded)
		return key, nil
	}

	keyPath := path + ".key"
	raw, err := os.ReadFile(keyPath)
	if err == nil {
		decoded, derr := hex.DecodeString(strings.TrimSpace(string(raw)))
		if derr != nil || len(decoded) != 32 {
			return key, fmt.Errorf("envstore: key file %s is not hex-encoded 32 bytes", keyPath)
		}
		copy(key[:], decoded)
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return key, fmt.Errorf("envstore: read key file: %w", err)
	}

	if _, err := rand.Read(key[:]); err != nil {
		return key, fmt.Errorf("envstore: generate key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return key, fmt.Errorf("envstore: create key dir: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(key[:])+"\n"), 0o600); err != nil {
		return key, fmt.Errorf("envstore: persist key file: %w", err)
	}
	return key, nil
}

func (s *Store) initFile() error {
	if _, err := os.Stat(s.path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("envstore: stat store: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("envstore: create store dir: %w", err)
	}
	if err := s.persist(envelope{Version: envelopeVersion, Secrets: map[string]item{}}); err != nil {
		return err
	}
	return nil
}

// Get decrypts the value for ref; a missing reference yields ErrNotFound.
func (s *Store) Get(ctx context.Context, ref string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	env, err := s.load()
	if err != nil {
		return "", err
	}
	it, ok := env.Secrets[ref]
	if !ok {
		return "", ports.ErrNotFound
	}
	return decryptValue(s.key, it)
}

// Put encrypts and persists value under ref, replacing any previous value.
func (s *Store) Put(ctx context.Context, ref, value string) error {
	if ref == "" {
		return errors.New("envstore: empty reference")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	env, err := s.load()
	if err != nil {
		return err
	}
	it, err := encryptValue(s.key, value)
	if err != nil {
		return err
	}
	env.Secrets[ref] = it
	return s.persist(env)
}

// Delete removes ref; deleting a missing ref is a no-op.
func (s *Store) Delete(ctx context.Context, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	env, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := env.Secrets[ref]; !ok {
		return nil
	}
	delete(env.Secrets, ref)
	return s.persist(env)
}

func (s *Store) load() (envelope, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return envelope{}, fmt.Errorf("envstore: read store: %w", err)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return envelope{}, fmt.Errorf("envstore: decode store: %w", err)
	}
	if env.Version != envelopeVersion {
		return envelope{}, fmt.Errorf("envstore: unsupported envelope version %d", env.Version)
	}
	if env.Secrets == nil {
		env.Secrets = map[string]item{}
	}
	return env, nil
}

func (s *Store) persist(env envelope) error {
	raw, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("envstore: encode store: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("envstore: write store: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("envstore: replace store: %w", err)
	}
	return nil
}

func encryptValue(key [32]byte, value string) (item, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return item{}, fmt.Errorf("envstore: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return item{}, fmt.Errorf("envstore: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return item{}, fmt.Errorf("envstore: nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, []byte(value), nil)
	return item{Nonce: base64.StdEncoding.EncodeToString(nonce), Cipher: base64.StdEncoding.EncodeToString(sealed)}, nil
}

func decryptValue(key [32]byte, it item) (string, error) {
	nonce, err := base64.StdEncoding.DecodeString(it.Nonce)
	if err != nil {
		return "", fmt.Errorf("envstore: decode nonce: %w", err)
	}
	sealed, err := base64.StdEncoding.DecodeString(it.Cipher)
	if err != nil {
		return "", fmt.Errorf("envstore: decode cipher: %w", err)
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("envstore: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("envstore: gcm: %w", err)
	}
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("envstore: decrypt (tampered or wrong key): %w", err)
	}
	return string(plain), nil
}