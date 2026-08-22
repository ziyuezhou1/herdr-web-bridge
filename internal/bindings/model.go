package bindings

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/protocol"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/security"
)

const SchemaVersion = 1

var (
	ErrBindingNotFound = errors.New("binding not found")
	ErrInvalidBinding  = errors.New("invalid binding")
)

type File struct {
	SchemaVersion int       `json:"schemaVersion"`
	NextSeq       uint64    `json:"nextSeq,omitempty"`
	Bindings      []Binding `json:"bindings"`
}

type Binding struct {
	ID                    string `json:"id"`
	ProjectPath           string `json:"projectPath"`
	ProjectLabel          string `json:"projectLabel"`
	URL                   string `json:"url"`
	URLMatch              string `json:"urlMatch"`
	PageTitle             string `json:"pageTitle"`
	Adapter               string `json:"adapter"`
	NotificationsEnabled  bool   `json:"notificationsEnabled"`
	CreatedAt             string `json:"createdAt"`
	UpdatedAt             string `json:"updatedAt"`
	LastState             string `json:"lastState"`
	LastEventID           string `json:"lastEventId"`
	LastNotifiedEventID   string `json:"lastNotifiedEventId,omitempty"`
	Seq                   uint64 `json:"seq"`
	SyncPending           bool   `json:"syncPending,omitempty"`
	QuickActionFile       string `json:"quickActionFile,omitempty"`
}

type NewBinding struct {
	ProjectPath          string
	ProjectLabel         string
	URL                  string
	PageTitle            string
	Adapter              string
	NotificationsEnabled bool
}

func Create(input NewBinding, now time.Time) (Binding, error) {
	id, err := newUUID()
	if err != nil {
		return Binding{}, err
	}
	projectPath, err := security.NormalizeWindowsPath(input.ProjectPath)
	if err != nil {
		return Binding{}, err
	}
	validatedURL, err := security.ValidateURL(input.URL, true)
	if err != nil {
		return Binding{}, err
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	binding := Binding{
		ID: id, ProjectPath: projectPath,
		ProjectLabel: security.TruncateRunes(input.ProjectLabel, 80),
		URL: validatedURL, URLMatch: "exact",
		PageTitle: security.TruncateRunes(input.PageTitle, 160),
		Adapter: input.Adapter, NotificationsEnabled: input.NotificationsEnabled,
		CreatedAt: stamp, UpdatedAt: stamp, LastState: "idle",
	}
	if err := Validate(binding); err != nil {
		return Binding{}, err
	}
	return binding, nil
}

func Validate(binding Binding) error {
	if !validUUID(binding.ID) {
		return fmt.Errorf("%w: invalid id", ErrInvalidBinding)
	}
	if _, err := security.NormalizeWindowsPath(binding.ProjectPath); err != nil {
		return fmt.Errorf("%w: project path", ErrInvalidBinding)
	}
	if _, err := security.ValidateURL(binding.URL, true); err != nil {
		return fmt.Errorf("%w: url", ErrInvalidBinding)
	}
	if binding.URLMatch != "exact" && binding.URLMatch != "normalized" {
		return fmt.Errorf("%w: urlMatch", ErrInvalidBinding)
	}
	if binding.Adapter != "chatgpt" && binding.Adapter != "custom-tool" && binding.Adapter != "generic" {
		return fmt.Errorf("%w: adapter", ErrInvalidBinding)
	}
	if !protocol.ValidateState(binding.LastState) {
		return fmt.Errorf("%w: lastState", ErrInvalidBinding)
	}
	if len([]rune(binding.ProjectLabel)) > 80 || len([]rune(binding.PageTitle)) > 160 || len(binding.LastEventID) > 128 || len(binding.LastNotifiedEventID) > 128 {
		return fmt.Errorf("%w: field too long", ErrInvalidBinding)
	}
	if _, err := time.Parse(time.RFC3339Nano, binding.CreatedAt); err != nil {
		return fmt.Errorf("%w: createdAt", ErrInvalidBinding)
	}
	if _, err := time.Parse(time.RFC3339Nano, binding.UpdatedAt); err != nil {
		return fmt.Errorf("%w: updatedAt", ErrInvalidBinding)
	}
	return nil
}

func NextSeq(binding *Binding) (uint64, error) {
	if binding.Seq == ^uint64(0) {
		return 0, errors.New("sequence exhausted")
	}
	binding.Seq++
	return binding.Seq, nil
}

func newUUID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(buf[0:4]), hex.EncodeToString(buf[4:6]), hex.EncodeToString(buf[6:8]), hex.EncodeToString(buf[8:10]), hex.EncodeToString(buf[10:16])), nil
}

func validUUID(value string) bool {
	parts := strings.Split(value, "-")
	if len(parts) != 5 || len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
		return false
	}
	_, err := hex.DecodeString(strings.Join(parts, ""))
	return err == nil
}
