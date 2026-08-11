package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
)

var ErrNotExist = errors.New("storage: file does not exist")

// Store — файловое JSON-хранилище в каталоге данных приложения.
type Store struct {
	root string
}

// Open открывает (или создаёт) каталог root как storage.
func Open(root string) (*Store, error) {
	if root == "" {
		return nil, fmt.Errorf("storage: empty root")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("storage: mkdir %s: %w", root, err)
	}
	return &Store{root: root}, nil
}

// OpenApp открывает storage в RootURI текущего Fyne-приложения.
func OpenApp() (*Store, error) {
	root, err := AppRoot()
	if err != nil {
		return nil, err
	}
	return Open(root)
}

// AppRoot возвращает путь к каталогу данных Fyne app storage.
func AppRoot() (string, error) {
	a := fyne.CurrentApp()
	if a == nil {
		return "", fmt.Errorf("storage: no current fyne app")
	}
	uri := a.Storage().RootURI()
	if uri == nil {
		return "", fmt.Errorf("storage: nil root uri")
	}
	path := uri.Path()
	if path == "" {
		return "", fmt.Errorf("storage: empty root path from %s", uri.String())
	}
	return path, nil
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) Path(name string) string {
	return filepath.Join(s.root, name)
}

func (s *Store) Exists(name string) bool {
	_, err := os.Stat(s.Path(name))
	return err == nil
}

// LoadJSON читает name в dst. Если файла нет — ErrNotExist.
func (s *Store) LoadJSON(name string, dst any) error {
	path := s.Path(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotExist
		}
		return fmt.Errorf("storage: read %s: %w", name, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("storage: decode %s: %w", name, err)
	}
	return nil
}

// SaveJSON атомарно пишет src в name (tmp + rename).
func (s *Store) SaveJSON(name string, src any) error {
	data, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: encode %s: %w", name, err)
	}
	data = append(data, '\n')

	path := s.Path(name)
	tmp, err := os.CreateTemp(s.root, name+".*.tmp")
	if err != nil {
		return fmt.Errorf("storage: temp for %s: %w", name, err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("storage: write temp %s: %w", name, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("storage: sync temp %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("storage: close temp %s: %w", name, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("storage: rename %s: %w", name, err)
	}
	return nil
}
