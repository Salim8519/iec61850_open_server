//go:build windows

package hostkey

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

// userRunKey is the real HKCU Run value; it needs no administrator rights.
type userRunKey struct{}

// SystemRunKey is the Run value of the current Windows user.
func SystemRunKey() RunKey { return userRunKey{} }

func (userRunKey) Get() (string, bool) {
	key, err := registry.OpenKey(registry.CURRENT_USER, RunKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer key.Close()
	value, _, err := key.GetStringValue(RunValueName)
	return value, err == nil
}

func (userRunKey) Set(command string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, RunKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(RunValueName, command)
}

func (userRunKey) Delete() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, RunKeyPath, registry.SET_VALUE)
	if err != nil {
		return nil // no Run key: nothing to remove
	}
	defer key.Close()
	if err := key.DeleteValue(RunValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}
