// Package bridge coordinates trusted storage, Herdr, Quick Actions, and transports.
package bridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/bindings"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/buildinfo"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/herdr"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/native"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/protocol"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/quickactions"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/security"
)

type Service struct {
	Store          *bindings.Store
	Herdr          *herdr.Client
	ExecutablePath string
	Writer         *native.Writer
	AllowedExtensionID string
	startedAt      time.Time
	mu             sync.RWMutex
	extensionID    string
	extensionSeen  time.Time
	retrying       map[string]bool
}

func New(store *bindings.Store, client *herdr.Client, executablePath string) *Service {
	return &Service{Store: store, Herdr: client, ExecutablePath: executablePath, AllowedExtensionID: buildinfo.ExtensionID, startedAt: time.Now(), retrying: make(map[string]bool)}
}

func (s *Service) HandleNative(req protocol.Request) protocol.Response {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if req.Type != "hello" {
		s.mu.RLock()
		authenticated := s.extensionID == s.AllowedExtensionID
		s.mu.RUnlock()
		if !authenticated {
			return failure(req, "hello_required", "extension must complete the native handshake first")
		}
	}
	switch req.Type {
	case "hello":
		var payload protocol.HelloPayload
		if err := protocol.DecodePayload(req.Payload, &payload); err != nil || payload.ExtensionID != s.AllowedExtensionID {
			return failure(req, "invalid_extension", "extension ID does not match the installed host")
		}
		s.mu.Lock()
		s.extensionID = payload.ExtensionID
		s.extensionSeen = time.Now()
		s.mu.Unlock()
		go s.SyncPending(context.Background())
		return success(req, map[string]interface{}{"bridgeVersion": buildinfo.Version, "hostName": buildinfo.HostName})
	case "ping":
		if err := validateEmptyPayload(req); err != nil { return failure(req, "invalid_payload", err.Error()) }
		return success(req, map[string]interface{}{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
	case "list_workspaces":
		if err := validateEmptyPayload(req); err != nil { return failure(req, "invalid_payload", err.Error()) }
		workspaces, err := s.Herdr.ListWorkspaces(ctx)
		if err != nil {
			return failure(req, "herdr_unavailable", err.Error())
		}
		return success(req, map[string]interface{}{"workspaces": herdr.Views(workspaces)})
	case "list_bindings":
		if err := validateEmptyPayload(req); err != nil { return failure(req, "invalid_payload", err.Error()) }
		file, err := s.Store.Load()
		if err != nil {
			return failure(req, "invalid_bindings", err.Error())
		}
		return success(req, file)
	case "binding_for_url":
		return s.bindingForURL(req)
	case "bind_page":
		return s.bindPage(ctx, req)
	case "unbind_page":
		return s.unbindPage(req)
	case "report_status":
		return s.reportStatus(ctx, req)
	case "focus_workspace":
		return s.focusWorkspace(ctx, req)
	default:
		return failure(req, "unsupported_type", "message type is not allowed")
	}
}

func (s *Service) HandleIPC(req protocol.Request) protocol.Response {
	switch req.Type {
	case "ping":
		if err := validateEmptyPayload(req); err != nil { return failure(req, "invalid_payload", err.Error()) }
		return success(req, map[string]string{"status": "ok"})
	case "status":
		if err := validateEmptyPayload(req); err != nil { return failure(req, "invalid_payload", err.Error()) }
		file, err := s.Store.Load()
		if err != nil {
			return failure(req, "invalid_bindings", err.Error())
		}
		s.mu.RLock()
		connected := s.extensionID != ""
		seen := s.extensionSeen
		s.mu.RUnlock()
		return success(req, map[string]interface{}{
			"extensionConnected": connected,
			"extensionLastSeen": seen.UTC().Format(time.RFC3339),
			"bindingCount": len(file.Bindings),
			"uptimeSeconds": int(time.Since(s.startedAt).Seconds()),
		})
	case "list_bindings":
		if err := validateEmptyPayload(req); err != nil { return failure(req, "invalid_payload", err.Error()) }
		file, err := s.Store.Load()
		if err != nil {
			return failure(req, "invalid_bindings", err.Error())
		}
		return success(req, file)
	case "open_binding":
		var payload protocol.BindingIDPayload
		if err := protocol.DecodePayload(req.Payload, &payload); err != nil {
			return failure(req, "invalid_payload", err.Error())
		}
		binding, err := s.Store.Get(payload.BindingID)
		if err != nil {
			return failure(req, "binding_not_found", "binding ID is not present in trusted configuration")
		}
		s.mu.RLock()
		connected := s.extensionID != ""
		s.mu.RUnlock()
		if !connected || s.Writer == nil {
			return failure(req, "extension_not_connected", "Edge extension is not connected")
		}
		event := protocol.Response{
			Version: protocol.Version,
			ID: "event-" + security.HashBindingID(binding.ID) + "-" + fmt.Sprint(time.Now().UnixNano()),
			Type: "open_binding",
			OK: true,
			Result: protocol.OpenBindingEvent{BindingID: binding.ID, URL: binding.URL, URLMatch: binding.URLMatch},
		}
		if err := s.Writer.Write(event); err != nil {
			return failure(req, "extension_write_failed", err.Error())
		}
		return success(req, map[string]string{"status": "focus_requested", "bindingId": binding.ID})
	default:
		return failure(req, "unsupported_type", "IPC command is not allowed")
	}
}

func (s *Service) bindingForURL(req protocol.Request) protocol.Response {
	var payload protocol.BindingForURLPayload
	if err := protocol.DecodePayload(req.Payload, &payload); err != nil {
		return failure(req, "invalid_payload", err.Error())
	}
	binding, err := s.Store.FindByURL(payload.URL)
	if errors.Is(err, bindings.ErrBindingNotFound) {
		return success(req, map[string]interface{}{"binding": nil})
	}
	if err != nil {
		return failure(req, "invalid_bindings", err.Error())
	}
	return success(req, map[string]interface{}{"binding": binding})
}

func (s *Service) bindPage(ctx context.Context, req protocol.Request) protocol.Response {
	var payload protocol.BindPagePayload
	if err := protocol.DecodePayload(req.Payload, &payload); err != nil {
		return failure(req, "invalid_payload", err.Error())
	}
	if len([]rune(payload.ProjectLabel)) > 80 || len([]rune(payload.PageTitle)) > 160 || (payload.Adapter != "chatgpt" && payload.Adapter != "custom-tool" && payload.Adapter != "generic") {
		return failure(req, "invalid_payload", "binding fields or adapter are invalid")
	}
	projectPath, err := security.ValidateExistingDirectory(payload.ProjectPath)
	if err != nil {
		return failure(req, "invalid_project_path", err.Error())
	}
	workspaces, err := s.Herdr.ListWorkspaces(ctx)
	if err != nil {
		return failure(req, "herdr_unavailable", err.Error())
	}
	workspace, err := herdr.ResolveWorkspace(workspaces, projectPath, payload.ProjectLabel)
	if err != nil {
		return failure(req, workspaceErrorCode(err), err.Error())
	}
	binding, err := bindings.Create(bindings.NewBinding{
		ProjectPath: projectPath, ProjectLabel: workspace.Label, URL: payload.URL,
		PageTitle: payload.PageTitle, Adapter: payload.Adapter,
		NotificationsEnabled: payload.NotificationsEnabled,
	}, time.Now())
	if err != nil {
		return failure(req, "invalid_binding", err.Error())
	}
	if err := s.Store.Add(binding); err != nil {
		return failure(req, "save_failed", err.Error())
	}
	quickActionGenerated := false
	quickActionReason := "herdr_plus_unavailable"
	if present, plusErr := s.Herdr.HasHerdrPlus(ctx); plusErr == nil && present {
		path, writeErr := quickactions.Write(binding, s.ExecutablePath)
		if writeErr == nil {
			updated, updateErr := s.Store.Update(binding.ID, func(current *bindings.Binding) error {
				current.QuickActionFile = path
				return nil
			})
			if updateErr == nil {
				binding = updated
				quickActionGenerated = true
				quickActionReason = ""
			} else {
				quickActionReason = "binding_update_failed"
			}
		} else {
			quickActionReason = "quick_action_write_failed"
		}
	}
	return success(req, map[string]interface{}{
		"binding": binding,
		"quickActionGenerated": quickActionGenerated,
		"quickActionReason": quickActionReason,
	})
}

func (s *Service) unbindPage(req protocol.Request) protocol.Response {
	var payload protocol.BindingIDPayload
	if err := protocol.DecodePayload(req.Payload, &payload); err != nil {
		return failure(req, "invalid_payload", err.Error())
	}
	binding, err := s.Store.Get(payload.BindingID)
	if err != nil {
		return failure(req, "binding_not_found", err.Error())
	}
	warning := ""
	if err := quickactions.Remove(binding); err != nil {
		warning = "quick_action_preserved: " + err.Error()
	}
	if _, err := s.Store.Remove(payload.BindingID); err != nil {
		return failure(req, "save_failed", err.Error())
	}
	return success(req, map[string]interface{}{"removed": true, "warning": warning})
}

func (s *Service) reportStatus(ctx context.Context, req protocol.Request) protocol.Response {
	var payload protocol.ReportStatusPayload
	if err := protocol.DecodePayload(req.Payload, &payload); err != nil {
		return failure(req, "invalid_payload", err.Error())
	}
	if !protocol.ValidateState(payload.State) || len(payload.EventID) > 128 || len([]rune(payload.PageTitle)) > 160 || len([]rune(payload.Reason)) > 160 {
		return failure(req, "invalid_status", "state or status fields are invalid")
	}
	previous, err := s.Store.Get(payload.BindingID)
	if err != nil {
		return failure(req, "binding_not_found", err.Error())
	}
	if payload.URL != "" && !URLsMatch(previous.URL, payload.URL) {
		return failure(req, "url_mismatch", "status URL does not match the trusted binding")
	}
	duplicateEvent := payload.EventID != "" && payload.EventID == previous.LastEventID && payload.State == previous.LastState
	if duplicateEvent {
		synced := !previous.SyncPending
		var syncError error
		if previous.SyncPending {
			synced, syncError = s.syncBinding(ctx, previous)
		}
		result := map[string]interface{}{
			"stored": true, "synced": synced, "duplicate": true,
			"fallbackNotification": false, "notification": map[string]string{},
		}
		if syncError != nil {
			result["syncError"] = security.TruncateRunes(syncError.Error(), 160)
			s.queueSyncRetry(previous.ID)
		}
		return success(req, result)
	}
	updated, err := s.Store.Advance(payload.BindingID, func(current *bindings.Binding) error {
		if payload.PageTitle != "" {
			current.PageTitle = security.TruncateRunes(payload.PageTitle, 160)
		}
		current.LastState = payload.State
		if payload.EventID != "" {
			current.LastEventID = payload.EventID
			if payload.NotificationHandled {
				current.LastNotifiedEventID = payload.EventID
			}
		}
		current.SyncPending = true
		return nil
	})
	if err != nil {
		return failure(req, "save_failed", err.Error())
	}
	synced, syncError := s.syncBinding(ctx, updated)
	shouldNotify := updated.NotificationsEnabled && payload.EventID != "" && updated.LastNotifiedEventID != payload.EventID && ((previous.LastState == "running" && payload.State == "done_unread") || payload.State == "error")
	fallbackNotification := false
	notificationError := ""
	if shouldNotify {
		// Persist the dedupe marker before the external side effect. This favors
		// at-most-once delivery if the process crashes between the two operations.
		_, markErr := s.Store.Update(updated.ID, func(current *bindings.Binding) error {
			current.LastNotifiedEventID = payload.EventID
			return nil
		})
		if markErr != nil {
			notificationError = "notification dedupe marker could not be persisted"
		} else {
			title, body, sound := notificationFor(updated, payload.State)
			notifyErr := s.Herdr.Notify(ctx, title, body, sound)
			fallbackNotification = notifyErr != nil
			if notifyErr != nil {
				notificationError = security.TruncateRunes(notifyErr.Error(), 160)
			}
		}
	}
	result := map[string]interface{}{
		"stored": true, "synced": synced,
		"fallbackNotification": fallbackNotification,
		"notification": map[string]string{},
	}
	if fallbackNotification {
		title, body, _ := notificationFor(updated, payload.State)
		result["notification"] = map[string]string{"title": title, "message": body}
	}
	if syncError != nil {
		result["syncError"] = security.TruncateRunes(syncError.Error(), 160)
		s.queueSyncRetry(updated.ID)
	}
	if notificationError != "" {
		result["notificationError"] = notificationError
	}
	return success(req, result)
}

func (s *Service) focusWorkspace(ctx context.Context, req protocol.Request) protocol.Response {
	var payload protocol.BindingIDPayload
	if err := protocol.DecodePayload(req.Payload, &payload); err != nil {
		return failure(req, "invalid_payload", err.Error())
	}
	binding, err := s.Store.Get(payload.BindingID)
	if err != nil {
		return failure(req, "binding_not_found", err.Error())
	}
	workspaces, err := s.Herdr.ListWorkspaces(ctx)
	if err != nil {
		return failure(req, "herdr_unavailable", err.Error())
	}
	workspace, err := herdr.ResolveWorkspace(workspaces, binding.ProjectPath, binding.ProjectLabel)
	if err != nil {
		return failure(req, workspaceErrorCode(err), err.Error())
	}
	if err := s.Herdr.FocusWorkspace(ctx, workspace.WorkspaceID); err != nil {
		return failure(req, "herdr_unavailable", err.Error())
	}
	return success(req, map[string]string{"workspaceId": workspace.WorkspaceID})
}

func (s *Service) syncBinding(ctx context.Context, binding bindings.Binding) (bool, error) {
	workspaces, err := s.Herdr.ListWorkspaces(ctx)
	if err != nil {
		return false, err
	}
	workspace, err := herdr.ResolveWorkspace(workspaces, binding.ProjectPath, binding.ProjectLabel)
	if err != nil {
		return false, err
	}
	title := binding.PageTitle
	if title == "" {
		title = binding.ProjectLabel
	}
	if err := s.Herdr.ReportMetadata(ctx, workspace.WorkspaceID, binding.LastState, title, binding.Seq); err != nil {
		return false, err
	}
	_, err = s.Store.Update(binding.ID, func(current *bindings.Binding) error {
		if current.Seq == binding.Seq {
			current.SyncPending = false
		}
		return nil
	})
	return err == nil, err
}

func (s *Service) SyncPending(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	file, err := s.Store.Load()
	if err != nil {
		return
	}
	for _, binding := range file.Bindings {
		if !binding.SyncPending {
			continue
		}
		synced, _ := s.syncBinding(ctx, binding)
		if !synced {
			s.queueSyncRetry(binding.ID)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (s *Service) queueSyncRetry(bindingID string) {
	s.mu.Lock()
	if s.retrying[bindingID] {
		s.mu.Unlock()
		return
	}
	s.retrying[bindingID] = true
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.retrying, bindingID)
			s.mu.Unlock()
		}()
		for _, delay := range []time.Duration{2 * time.Second, 5 * time.Second, 15 * time.Second} {
			time.Sleep(delay)
			binding, err := s.Store.Get(bindingID)
			if err != nil || !binding.SyncPending {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			synced, _ := s.syncBinding(ctx, binding)
			cancel()
			if synced {
				return
			}
		}
	}()
}

func URLsMatch(trusted, observed string) bool {
	if trusted == observed {
		return true
	}
	left, leftErr := security.NormalizeURL(trusted)
	right, rightErr := security.NormalizeURL(observed)
	return leftErr == nil && rightErr == nil && left == right
}

func notificationFor(binding bindings.Binding, state string) (string, string, string) {
	if state == "error" {
		return "网页任务出错", security.TruncateRunes(binding.ProjectLabel+"：生成失败，请返回网页检查", 240), "request"
	}
	return "网页任务已完成", security.TruncateRunes(binding.ProjectLabel+"："+adapterLabel(binding.Adapter)+" 已生成结果，等待查看", 240), "done"
}

func adapterLabel(adapter string) string {
	switch adapter {
	case "chatgpt":
		return "ChatGPT"
	case "custom-tool":
		return "网页工具"
	default:
		return "网页"
	}
}

func workspaceErrorCode(err error) string {
	switch {
	case errors.Is(err, herdr.ErrWorkspaceAmbiguous):
		return "ambiguous_workspace"
	case errors.Is(err, herdr.ErrPathUnavailable):
		return "path_unavailable"
	default:
		return "workspace_not_found"
	}
}

func success(req protocol.Request, result interface{}) protocol.Response {
	return protocol.Success(req.ID, req.Type, result)
}

func failure(req protocol.Request, code, message string) protocol.Response {
	return protocol.Failure(req.ID, req.Type, code, security.TruncateRunes(security.RedactLog(message), 240))
}

func validateEmptyPayload(req protocol.Request) error {
	var empty struct{}
	return protocol.DecodePayload(req.Payload, &empty)
}

func ExecutablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) ValidateOrigin(origin string) bool {
	return strings.TrimSuffix(origin, "/") == "chrome-extension://"+s.AllowedExtensionID
}
