package bindings

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testBinding(t *testing.T, projectPath string) Binding {
	t.Helper()
	binding, err := Create(NewBinding{
		ProjectPath: projectPath, ProjectLabel: "VirtualDNA",
		URL: "https://chatgpt.com/c/example?private=yes", PageTitle: "Run 1",
		Adapter: "chatgpt", NotificationsEnabled: true,
	}, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func TestAtomicSaveBackupAndBindingLookup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "bindings.json")
	store := NewStore(path)
	binding := testBinding(t, t.TempDir())
	if err := store.Add(binding); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(binding.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(binding.ID, func(current *Binding) error {
		current.LastState = "running"
		_, err := NextSeq(current)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	if err := ValidateReader(bytes.NewReader(backup)); err != nil {
		t.Fatalf("backup is not a valid binding file: %v", err)
	}
	loaded, err := store.Load()
	if err != nil || loaded.Bindings[0].LastState != "running" || loaded.Bindings[0].Seq != 1 {
		t.Fatalf("unexpected updated file: %#v, %v", loaded, err)
	}
}

func TestSequenceIsStrictlyMonotonic(t *testing.T) {
	binding := testBinding(t, t.TempDir())
	for want := uint64(1); want <= 3; want++ {
		got, err := NextSeq(&binding)
		if err != nil || got != want {
			t.Fatalf("want %d, got %d (%v)", want, got, err)
		}
	}
}

func TestStoredSequenceIsMonotonicAcrossBindings(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "bindings.json"))
	one := testBinding(t, t.TempDir())
	two := testBinding(t, t.TempDir())
	if err := store.Add(one); err != nil { t.Fatal(err) }
	if err := store.Add(two); err != nil { t.Fatal(err) }
	first, err := store.Advance(two.ID, func(*Binding) error { return nil })
	if err != nil { t.Fatal(err) }
	second, err := store.Advance(one.ID, func(*Binding) error { return nil })
	if err != nil { t.Fatal(err) }
	if second.Seq <= first.Seq {
		t.Fatalf("sequence regressed across bindings: %d then %d", first.Seq, second.Seq)
	}
}
