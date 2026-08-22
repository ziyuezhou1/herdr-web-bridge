// Package protocol defines the only messages accepted across extension and IPC boundaries.
package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	Version              = 1
	MaxNativeMessageSize = 1 << 20
	MaxIPCMessageSize    = 64 << 10
)

var (
	ErrInvalidEnvelope = errors.New("invalid protocol envelope")
	ErrUnsupportedType = errors.New("unsupported message type")
)

type Request struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	Version int         `json:"version"`
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	OK      bool        `json:"ok"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type HelloPayload struct {
	ExtensionID string `json:"extensionId"`
}

type BindPagePayload struct {
	ProjectPath          string `json:"projectPath"`
	ProjectLabel         string `json:"projectLabel"`
	URL                  string `json:"url"`
	PageTitle            string `json:"pageTitle"`
	Adapter              string `json:"adapter"`
	NotificationsEnabled bool   `json:"notificationsEnabled"`
}

type BindingIDPayload struct {
	BindingID string `json:"bindingId"`
}

type BindingForURLPayload struct {
	URL string `json:"url"`
}

type ReportStatusPayload struct {
	BindingID string `json:"bindingId"`
	State     string `json:"state"`
	EventID   string `json:"eventId,omitempty"`
	PageTitle string `json:"pageTitle,omitempty"`
	URL       string `json:"url,omitempty"`
	Reason    string `json:"reason,omitempty"`
	NotificationHandled bool `json:"notificationHandled,omitempty"`
}

type OpenBindingEvent struct {
	BindingID string `json:"bindingId"`
	URL       string `json:"url"`
	URLMatch  string `json:"urlMatch"`
}

func DecodeRequest(data []byte, allowed map[string]struct{}) (Request, error) {
	if len(data) == 0 || len(data) > MaxNativeMessageSize || !utf8.Valid(data) {
		return Request{}, ErrInvalidEnvelope
	}
	var req Request
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return Request{}, fmt.Errorf("%w: %v", ErrInvalidEnvelope, err)
	}
	var trailing interface{}
	if err := dec.Decode(&trailing); err != io.EOF {
		return Request{}, ErrInvalidEnvelope
	}
	if req.Version != Version || req.ID == "" || len(req.ID) > 128 || req.Type == "" || len(req.Type) > 64 {
		return Request{}, ErrInvalidEnvelope
	}
	if _, ok := allowed[req.Type]; !ok {
		return Request{}, ErrUnsupportedType
	}
	return req, nil
}

func DecodePayload(raw json.RawMessage, target interface{}) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}
	return nil
}

func Success(id, messageType string, result interface{}) Response {
	return Response{Version: Version, ID: id, Type: messageType, OK: true, Result: result}
}

func Failure(id, messageType, code, message string) Response {
	return Response{Version: Version, ID: id, Type: messageType, OK: false, Error: &ErrorBody{Code: code, Message: message}}
}

var NativeRequestTypes = map[string]struct{}{
	"hello": {}, "ping": {}, "list_workspaces": {}, "list_bindings": {},
	"binding_for_url": {}, "bind_page": {}, "unbind_page": {},
	"report_status": {}, "focus_workspace": {},
}

var IPCRequestTypes = map[string]struct{}{
	"ping": {}, "status": {}, "list_bindings": {}, "open_binding": {},
}

var States = map[string]struct{}{
	"idle": {}, "running": {}, "done_unread": {}, "viewed": {}, "error": {}, "unknown": {},
}

func ValidateState(state string) bool {
	_, ok := States[state]
	return ok
}
