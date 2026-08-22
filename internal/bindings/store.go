package bindings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/fileutil"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/security"
)

type Store struct {
	path string
	mu   sync.Mutex
}

func DefaultPath() (string, error) {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	return filepath.Join(root, "HerdrWebBridge", "bindings.json"), nil
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string { return s.path }

func (s *Store) Load() (File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked()
}

func (s *Store) loadUnlocked() (File, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return File{SchemaVersion: SchemaVersion, Bindings: []Binding{}}, nil
	}
	if err != nil {
		return File{}, err
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return File{}, fmt.Errorf("parse bindings: %w", err)
	}
	if file.SchemaVersion != SchemaVersion {
		return File{}, fmt.Errorf("unsupported bindings schema version %d", file.SchemaVersion)
	}
	seen := make(map[string]struct{}, len(file.Bindings))
	for i := range file.Bindings {
		if err := Validate(file.Bindings[i]); err != nil {
			return File{}, fmt.Errorf("binding %d: %w", i, err)
		}
		if _, exists := seen[file.Bindings[i].ID]; exists {
			return File{}, fmt.Errorf("duplicate binding id")
		}
		seen[file.Bindings[i].ID] = struct{}{}
	}
	return file, nil
}

func (s *Store) Save(file File) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(file)
}

func (s *Store) saveUnlocked(file File) error {
	file.SchemaVersion = SchemaVersion
	if file.Bindings == nil {
		file.Bindings = []Binding{}
	}
	for _, binding := range file.Bindings {
		if err := Validate(binding); err != nil {
			return err
		}
	}
	sort.SliceStable(file.Bindings, func(i, j int) bool { return file.Bindings[i].CreatedAt < file.Bindings[j].CreatedAt })
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if current, err := os.ReadFile(s.path); err == nil {
		if err := writeSyncedFile(s.path+".bak.tmp", current); err != nil {
			return fmt.Errorf("write backup: %w", err)
		}
		if err := fileutil.Replace(s.path+".bak.tmp", s.path+".bak"); err != nil {
			return fmt.Errorf("replace backup: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temp, err := os.CreateTemp(dir, "bindings-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temp.Write(data); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := fileutil.Replace(tempPath, s.path); err != nil {
		return err
	}
	committed = true
	return nil
}

func writeSyncedFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func (s *Store) Add(binding Binding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	for _, existing := range file.Bindings {
		if existing.ID == binding.ID {
			return fmt.Errorf("duplicate binding id")
		}
	}
	file.Bindings = append(file.Bindings, binding)
	return s.saveUnlocked(file)
}

func (s *Store) Get(id string) (Binding, error) {
	file, err := s.Load()
	if err != nil {
		return Binding{}, err
	}
	for _, binding := range file.Bindings {
		if binding.ID == id {
			return binding, nil
		}
	}
	return Binding{}, ErrBindingNotFound
}

func (s *Store) FindByURL(rawURL string) (Binding, error) {
	file, err := s.Load()
	if err != nil {
		return Binding{}, err
	}
	for _, binding := range file.Bindings {
		if binding.URL == rawURL {
			return binding, nil
		}
	}
	normalized, err := security.NormalizeURL(rawURL)
	if err != nil {
		return Binding{}, ErrBindingNotFound
	}
	for _, binding := range file.Bindings {
		candidate, candidateErr := security.NormalizeURL(binding.URL)
		if candidateErr == nil && candidate == normalized {
			return binding, nil
		}
	}
	return Binding{}, ErrBindingNotFound
}

func (s *Store) Update(id string, update func(*Binding) error) (Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := s.loadUnlocked()
	if err != nil {
		return Binding{}, err
	}
	for i := range file.Bindings {
		if file.Bindings[i].ID != id {
			continue
		}
		if err := update(&file.Bindings[i]); err != nil {
			return Binding{}, err
		}
		file.Bindings[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := s.saveUnlocked(file); err != nil {
			return Binding{}, err
		}
		return file.Bindings[i], nil
	}
	return Binding{}, ErrBindingNotFound
}

// Advance updates a binding and assigns a sequence that is monotonic across the
// fixed Herdr metadata source as well as within the binding itself.
func (s *Store) Advance(id string, update func(*Binding) error) (Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := s.loadUnlocked()
	if err != nil {
		return Binding{}, err
	}
	for _, binding := range file.Bindings {
		if binding.Seq > file.NextSeq {
			file.NextSeq = binding.Seq
		}
	}
	if file.NextSeq == ^uint64(0) {
		return Binding{}, errors.New("sequence exhausted")
	}
	for i := range file.Bindings {
		if file.Bindings[i].ID != id {
			continue
		}
		file.NextSeq++
		file.Bindings[i].Seq = file.NextSeq
		if err := update(&file.Bindings[i]); err != nil {
			return Binding{}, err
		}
		file.Bindings[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := s.saveUnlocked(file); err != nil {
			return Binding{}, err
		}
		return file.Bindings[i], nil
	}
	return Binding{}, ErrBindingNotFound
}

func (s *Store) Remove(id string) (Binding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := s.loadUnlocked()
	if err != nil {
		return Binding{}, err
	}
	for i := range file.Bindings {
		if file.Bindings[i].ID == id {
			removed := file.Bindings[i]
			file.Bindings = append(file.Bindings[:i], file.Bindings[i+1:]...)
			return removed, s.saveUnlocked(file)
		}
	}
	return Binding{}, ErrBindingNotFound
}

func ValidateReader(reader io.Reader) error {
	data, err := io.ReadAll(io.LimitReader(reader, 10<<20))
	if err != nil {
		return err
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}
	if file.SchemaVersion != SchemaVersion {
		return errors.New("unsupported schema")
	}
	for _, binding := range file.Bindings {
		if err := Validate(binding); err != nil {
			return err
		}
	}
	return nil
}
