package codexbridge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Profile is one AI account login: the laptop's default CLI login ("system")
// or an isolated CODEX_HOME / CLAUDE_CONFIG_DIR created by the bridge.
type Profile struct {
	Provider, ID, Label, Dir string
	System                   bool
}

var accountLabels = map[string]string{ProviderCodex: "ChatGPT", ProviderClaude: "Claude"}

type registryEntry struct {
	ID       string    `json:"id"`
	Provider string    `json:"provider"`
	Label    string    `json:"label"`
	Created  time.Time `json:"created_at"`
}
type registryFile struct {
	Version  int               `json:"version"`
	Active   map[string]string `json:"active"`
	Accounts []registryEntry   `json:"accounts"`
}

// Registry persists extra account profiles and the active selection (0700 dirs, 0600 file).
type Registry struct {
	Root string
	mu   sync.Mutex
	data registryFile
}

// DefaultRegistryRoot is $XDG_DATA_HOME/portway/bridge (default ~/.local/share).
func DefaultRegistryRoot() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if !filepath.IsAbs(base) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "portway", "bridge"), nil
}
func OpenRegistry(root string) (*Registry, error) {
	if !filepath.IsAbs(root) {
		return nil, errors.New("bridge data directory must be absolute")
	}
	r := &Registry{Root: root, data: registryFile{Version: 1, Active: map[string]string{}, Accounts: []registryEntry{}}}
	for _, p := range []string{ProviderCodex, ProviderClaude} {
		if err := os.MkdirAll(filepath.Join(root, "accounts", p), 0700); err != nil {
			return nil, err
		}
	}
	for _, dir := range []string{root, filepath.Join(root, "accounts"), filepath.Join(root, "accounts", ProviderCodex), filepath.Join(root, "accounts", ProviderClaude)} {
		if err := os.Chmod(dir, 0700); err != nil {
			return nil, err
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, "accounts.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		var data registryFile
		if json.Unmarshal(raw, &data) != nil || data.Version != 1 {
			return nil, errors.New("bridge account registry is unreadable")
		}
		for _, e := range data.Accounts {
			if ValidProvider(e.Provider) && ValidAccountID(e.Provider, e.ID) && plain(e.Label, 60) && e.Label != "" {
				if st, err := os.Stat(r.dir(e.Provider, e.ID)); err == nil && st.IsDir() {
					r.data.Accounts = append(r.data.Accounts, e)
				}
			}
		}
		for p, id := range data.Active {
			if _, ok := r.lookup(p, id); ok {
				r.data.Active[p] = id
			}
		}
	}
	// Remove profile dirs left by an interrupted login (never registered).
	for _, p := range []string{ProviderCodex, ProviderClaude} {
		entries, _ := os.ReadDir(filepath.Join(root, "accounts", p))
		for _, e := range entries {
			if _, ok := r.lookup(p, e.Name()); !ok && ValidAccountID(p, e.Name()) {
				_ = os.RemoveAll(r.dir(p, e.Name()))
			}
		}
	}
	return r, r.save()
}
func (r *Registry) dir(provider, id string) string {
	return filepath.Join(r.Root, "accounts", provider, id)
}
func (r *Registry) system(provider string) Profile {
	return Profile{Provider: provider, ID: provider + "-system", Label: accountLabels[provider], System: true}
}
func (r *Registry) lookup(provider, id string) (Profile, bool) {
	if id == provider+"-system" {
		return r.system(provider), true
	}
	for _, e := range r.data.Accounts {
		if e.Provider == provider && e.ID == id {
			return Profile{Provider: provider, ID: id, Label: e.Label, Dir: r.dir(provider, id)}, true
		}
	}
	return Profile{}, false
}
func (r *Registry) Get(provider, id string) (Profile, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lookup(provider, id)
}
func (r *Registry) Profiles(provider string) []Profile {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Profile{r.system(provider)}
	for _, e := range r.data.Accounts {
		if e.Provider == provider {
			out = append(out, Profile{Provider: provider, ID: e.ID, Label: e.Label, Dir: r.dir(provider, e.ID)})
		}
	}
	return out
}
func (r *Registry) Active(provider string) Profile {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.lookup(provider, r.data.Active[provider]); ok {
		return p
	}
	return r.system(provider)
}

// NewPending creates a fresh, unregistered profile directory for one login attempt.
func (r *Registry) NewPending(provider string) (Profile, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return Profile{}, err
	}
	id := provider + "-" + hex.EncodeToString(b)
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 1
	for _, e := range r.data.Accounts {
		if e.Provider == provider {
			if i, err := strconv.Atoi(strings.TrimPrefix(e.Label, accountLabels[provider]+" ")); err == nil && i > n {
				n = i
			}
		}
	}
	p := Profile{Provider: provider, ID: id, Label: fmt.Sprintf("%s %d", accountLabels[provider], n+1), Dir: r.dir(provider, id)}
	return p, os.Mkdir(p.Dir, 0700)
}

// Commit registers a successfully logged-in pending profile and makes it active.
func (r *Registry) Commit(p Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lookup(p.Provider, p.ID); !ok {
		r.data.Accounts = append(r.data.Accounts, registryEntry{ID: p.ID, Provider: p.Provider, Label: p.Label, Created: time.Now().UTC()})
	}
	r.data.Active[p.Provider] = p.ID
	return r.save()
}

// Discard deletes an unregistered profile directory (failed or cancelled login).
func (r *Registry) Discard(p Profile) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lookup(p.Provider, p.ID); !ok && !p.System && ValidAccountID(p.Provider, p.ID) && p.Dir == r.dir(p.Provider, p.ID) {
		_ = os.RemoveAll(p.Dir)
	}
}
func (r *Registry) Activate(provider, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lookup(provider, id); !ok {
		return errors.New("AI_ACCOUNT_NOT_FOUND")
	}
	r.data.Active[provider] = id
	return r.save()
}

// Remove forgets an extra profile and deletes its directory; the system login is never removed.
func (r *Registry) Remove(provider, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.lookup(provider, id)
	if !ok {
		return errors.New("AI_ACCOUNT_NOT_FOUND")
	}
	if p.System {
		return errors.New("AI_ACCOUNT_SYSTEM")
	}
	kept := r.data.Accounts[:0]
	for _, e := range r.data.Accounts {
		if e.Provider != provider || e.ID != id {
			kept = append(kept, e)
		}
	}
	r.data.Accounts = kept
	if r.data.Active[provider] == id {
		delete(r.data.Active, provider)
	}
	if err := r.save(); err != nil {
		return err
	}
	return os.RemoveAll(p.Dir)
}
func (r *Registry) save() error {
	raw, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(r.Root, ".accounts-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(raw)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(r.Root, "accounts.json"))
}
