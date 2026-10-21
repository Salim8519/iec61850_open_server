package adminauth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// AccountPath is the owner login file inside the data folder. It is separate from
// the license database so the setup command can reset it while the server runs.
func AccountPath(dataDir string) string { return filepath.Join(dataDir, "admin-account.json") }

// Keeper reads and writes the account file one change at a time.
type Keeper struct {
	path string
	mu   sync.Mutex
}

// NewKeeper returns the keeper for dataDir.
func NewKeeper(dataDir string) *Keeper { return &Keeper{path: AccountPath(dataDir)} }

// Load returns the account and whether setup has been run.
func (k *Keeper) Load() (Account, bool, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.load()
}

func (k *Keeper) load() (Account, bool, error) {
	var account Account
	raw, err := os.ReadFile(k.path)
	if errors.Is(err, os.ErrNotExist) {
		return account, false, nil
	}
	if err != nil {
		return account, false, err
	}
	if err := json.Unmarshal(raw, &account); err != nil {
		return account, false, errors.New("admin-account.json is damaged; run the admin setup again")
	}
	return account, account.PasswordHash != "" && account.TOTPSecret != "", nil
}

// Update loads the account, applies change and saves it when change returns true.
func (k *Keeper) Update(change func(*Account) bool) (Account, bool, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	account, ready, err := k.load()
	if err != nil || !ready {
		return account, ready, err
	}
	if change(&account) {
		err = k.save(account)
	}
	return account, ready, err
}

// Replace writes a brand-new account (setup or reset).
func (k *Keeper) Replace(account Account) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.save(account)
}

// save writes a temporary file and renames it, so a crash never leaves half a file.
func (k *Keeper) save(account Account) error {
	raw, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(k.path), 0o700); err != nil {
		return err
	}
	temp := k.path + ".tmp"
	if err := os.WriteFile(temp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(temp, k.path)
}
