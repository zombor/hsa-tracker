package receipt

import (
	"fmt"
	"os"
	"path/filepath"
)

// Storage defines the interface for file storage operations
type Storage interface {
	// Save saves a file and returns the path/filename
	Save(filename string, data []byte) (string, error)

	// Get retrieves a file by path
	Get(path string) ([]byte, error)

	// Delete removes a file
	Delete(path string) error
}

// LocalStorage implements the Storage interface using local filesystem
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new LocalStorage instance
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
	}, nil
}

// Save saves a file to local storage
func (l *LocalStorage) Save(filename string, data []byte) (string, error) {
	path := filepath.Join(l.basePath, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	return filename, nil
}

// Get retrieves a file from local storage
func (l *LocalStorage) Get(path string) ([]byte, error) {
	fullPath := filepath.Join(l.basePath, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return data, nil
}

// Delete removes a file from local storage
func (l *LocalStorage) Delete(path string) error {
	fullPath := filepath.Join(l.basePath, path)
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}
	return nil
}

