package ipc

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/native"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/protocol"
)

func TestMockIPCAllowsPingAndRejectsNoArbitraryCommand(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	go serveConnection(server, func(req protocol.Request) protocol.Response {
		return protocol.Success(req.ID, req.Type, map[string]string{"status": "ok"})
	})
	payload, _ := json.Marshal(map[string]interface{}{})
	req := protocol.Request{Version: 1, ID: "ipc-test", Type: "ping", Payload: payload}
	if err := native.WriteFrame(client, req, protocol.MaxIPCMessageSize); err != nil {
		t.Fatal(err)
	}
	data, err := native.ReadFrame(client, protocol.MaxIPCMessageSize)
	if err != nil {
		t.Fatal(err)
	}
	var response protocol.Response
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.ID != req.ID {
		t.Fatalf("unexpected IPC response: %#v", response)
	}

	if _, err := protocol.DecodeRequest([]byte(`{"version":1,"id":"bad","type":"open_path","payload":{"path":"C:\\"}}`), protocol.IPCRequestTypes); err == nil {
		t.Fatal("IPC must reject arbitrary path commands")
	}
}

