// Package ipc exposes a small per-user command allowlist over a Windows named pipe.
package ipc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os/user"
	"time"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/native"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/protocol"
)

func PipeIdentity() (name string, sid string, err error) {
	current, err := user.Current()
	if err != nil || current.Uid == "" {
		return "", "", errors.New("cannot determine current Windows user SID")
	}
	// SID characters are safe in pipe names and make ownership auditable.
	return `\\.\pipe\herdr-web-bridge-` + current.Uid, current.Uid, nil
}

type Handler func(protocol.Request) protocol.Response

func serveConnection(connection io.ReadWriteCloser, handler Handler) {
	defer connection.Close()
	for {
		data, err := native.ReadFrame(connection, protocol.MaxIPCMessageSize)
		if err != nil {
			return
		}
		req, err := protocol.DecodeRequest(data, protocol.IPCRequestTypes)
		if err != nil {
			_ = native.WriteFrame(connection, protocol.Failure("", "error", "invalid_message", err.Error()), protocol.MaxIPCMessageSize)
			continue
		}
		if err := native.WriteFrame(connection, handler(req), protocol.MaxIPCMessageSize); err != nil {
			return
		}
	}
}

func Call(messageType string, payload interface{}, timeout time.Duration) (protocol.Response, error) {
	if _, ok := protocol.IPCRequestTypes[messageType]; !ok {
		return protocol.Response{}, protocol.ErrUnsupportedType
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return protocol.Response{}, err
	}
	id, err := randomID()
	if err != nil {
		return protocol.Response{}, err
	}
	connection, err := dial(timeout)
	if err != nil {
		return protocol.Response{}, err
	}
	defer connection.Close()
	req := protocol.Request{Version: protocol.Version, ID: id, Type: messageType, Payload: payloadJSON}
	if err := native.WriteFrame(connection, req, protocol.MaxIPCMessageSize); err != nil {
		return protocol.Response{}, err
	}
	data, err := native.ReadFrame(connection, protocol.MaxIPCMessageSize)
	if err != nil {
		return protocol.Response{}, err
	}
	var response protocol.Response
	if err := json.Unmarshal(data, &response); err != nil {
		return protocol.Response{}, err
	}
	if response.Version != protocol.Version || response.ID != id {
		return protocol.Response{}, errors.New("mismatched IPC response")
	}
	return response, nil
}

func randomID() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

