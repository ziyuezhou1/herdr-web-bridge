package herdr

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/security"
)

type recordingRunner struct {
	output []byte
	err    error
	calls  [][]string
}

func (r *recordingRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{}, args...))
	return r.output, r.err
}

type commandResult struct {
	output []byte
	err    error
}

type commandRunner struct {
	responses map[string]commandResult
	calls     []string
}

func (r *commandRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	command := strings.Join(args, " ")
	r.calls = append(r.calls, command)
	result, ok := r.responses[command]
	if !ok {
		return nil, errors.New("unexpected command: " + command)
	}
	return result.output, result.err
}

func TestWorkspaceParsingAndResolution(t *testing.T) {
	data := []byte(`{"workspaces":[{"workspace_id":"w1","label":"Other","focused":false,"worktree":{"checkout_path":"T:\\VirtualDNA"}},{"workspace_id":"w2","label":"VirtualDNA","focused":false,"worktree":{"checkout_path":"t:\\virtualdna\\"}},{"workspace_id":"w3","label":"No path","focused":true,"worktree":null}]}`)
	workspaces, err := ParseWorkspaces(data)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveWorkspace(workspaces, `T:\VirtualDNA`, "VirtualDNA")
	if err != nil || resolved.WorkspaceID != "w2" {
		t.Fatalf("label tie-break failed: %#v, %v", resolved, err)
	}
	workspaces[0].Focused = true
	resolved, err = ResolveWorkspace(workspaces, `T:\VirtualDNA`, "VirtualDNA")
	if err != nil || resolved.WorkspaceID != "w1" {
		t.Fatalf("focused tie-break failed: %#v, %v", resolved, err)
	}
	views := Views(workspaces)
	if views[2].PathAvailable {
		t.Fatal("workspace without a public checkout path must not be bindable")
	}
}

func TestWorkspaceConflictIsNotSilentlySelected(t *testing.T) {
	workspaces := []Workspace{
		{WorkspaceID: "one", Label: "Same", Worktree: &WorkspaceWorktree{CheckoutPath: `T:\Same`}},
		{WorkspaceID: "two", Label: "Same", Worktree: &WorkspaceWorktree{CheckoutPath: `t:\same`}},
	}
	_, err := ResolveWorkspace(workspaces, `T:\Same`, "Same")
	if !errors.Is(err, ErrWorkspaceAmbiguous) {
		t.Fatalf("expected ambiguity, got %v", err)
	}
}

func TestListWorkspacesUsesUniquePaneCWDForPlainWorkspaces(t *testing.T) {
	runner := &commandRunner{responses: map[string]commandResult{
		"workspace list": {output: []byte(`{"workspaces":[{"workspace_id":"git","label":"Git","worktree":{"checkout_path":"T:\\GitProject"}},{"workspace_id":"plain","label":"Plain","worktree":null},{"workspace_id":"same","label":"Same","worktree":null},{"workspace_id":"ambiguous","label":"Ambiguous","worktree":null}]}`)},
		"pane list":      {output: []byte(`{"result":{"panes":[{"workspace_id":"plain","pane_id":"p1","cwd":"D:\\PlainProject"},{"workspace_id":"same","pane_id":"p2","cwd":"d:\\SameProject\\"},{"workspace_id":"same","pane_id":"p3","cwd":"D:\\sameproject"},{"workspace_id":"ambiguous","pane_id":"p4","cwd":"D:\\One"},{"workspace_id":"ambiguous","pane_id":"p5","cwd":"D:\\Two"}]}}`)},
	}}
	workspaces, err := NewClient(runner).ListWorkspaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runner.calls, []string{"workspace list", "pane list"}) {
		t.Fatalf("unexpected Herdr calls: %#v", runner.calls)
	}
	views := Views(workspaces)
	if !views[0].PathAvailable || views[0].ProjectPath != `T:\GitProject` || views[0].PathSource != PathSourceWorktree {
		t.Fatalf("worktree path should remain authoritative: %#v", views[0])
	}
	if !views[1].PathAvailable || views[1].ProjectPath != `D:\PlainProject` || views[1].PathSource != PathSourcePaneCWD {
		t.Fatalf("unique pane cwd was not exposed: %#v", views[1])
	}
	if !views[2].PathAvailable || !security.SameWindowsPath(views[2].ProjectPath, `D:\SameProject`) {
		t.Fatalf("equivalent pane paths should be treated as one path: %#v", views[2])
	}
	if views[3].PathAvailable || views[3].PathReason != PathReasonAmbiguousPaneCWD {
		t.Fatalf("different pane paths must remain unavailable: %#v", views[3])
	}
	resolved, err := ResolveWorkspace(workspaces, `D:\PlainProject`, "Plain")
	if err != nil || resolved.WorkspaceID != "plain" {
		t.Fatalf("pane-derived path was not used during re-resolution: %#v (%v)", resolved, err)
	}
}

func TestListWorkspacesKeepsPathlessEntriesWhenPaneListFails(t *testing.T) {
	runner := &commandRunner{responses: map[string]commandResult{
		"workspace list": {output: []byte(`[{"workspace_id":"plain","label":"Plain","worktree":null}]`)},
		"pane list":      {err: errors.New("pane API unavailable")},
	}}
	workspaces, err := NewClient(runner).ListWorkspaces(context.Background())
	if err != nil {
		t.Fatalf("workspace discovery should degrade instead of failing: %v", err)
	}
	views := Views(workspaces)
	if len(views) != 1 || views[0].PathAvailable || views[0].PathReason != PathReasonPaneListUnavailable {
		t.Fatalf("unexpected degraded workspace view: %#v", views)
	}
}

func TestListWorkspacesSkipsPaneListWhenWorktreePathsExist(t *testing.T) {
	runner := &commandRunner{responses: map[string]commandResult{
		"workspace list": {output: []byte(`[{"workspace_id":"git","label":"Git","worktree":{"checkout_path":"T:\\GitProject"}}]`)},
	}}
	workspaces, err := NewClient(runner).ListWorkspaces(context.Background())
	if err != nil || len(workspaces) != 1 {
		t.Fatalf("unexpected workspace result: %#v (%v)", workspaces, err)
	}
	if !reflect.DeepEqual(runner.calls, []string{"workspace list"}) {
		t.Fatalf("pane list should not be called: %#v", runner.calls)
	}
}

func TestMetadataAndNotificationArguments(t *testing.T) {
	args, err := MetadataArgs("workspace-1", "done_unread", "A very small page", 7)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"workspace", "report-metadata", "workspace-1", "--source", Source, "--token", "web_status=✅ 等待查看：A very small page", "--seq", "7", "--ttl-ms", "86400000"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("unexpected metadata args:\nwant %#v\ngot  %#v", want, args)
	}
	clear, err := MetadataArgs("workspace-1", "viewed", "ignored", 8)
	if err != nil || strings.Join(clear, " ") != "workspace report-metadata workspace-1 --source herdr-web-bridge --clear-token web_status --seq 8" {
		t.Fatalf("unexpected clear args: %#v (%v)", clear, err)
	}
	notification, err := NotificationArgs("网页任务已完成", "Project：done", "done")
	if err != nil || notification[0] != "notification" || notification[len(notification)-1] != "done" {
		t.Fatalf("unexpected notification args: %#v (%v)", notification, err)
	}
}

func TestReportMetadataUsesBoundedRetry(t *testing.T) {
	runner := &recordingRunner{err: errors.New("offline")}
	client := NewClient(runner)
	err := client.ReportMetadata(context.Background(), "w", "running", "Page", 1)
	if err == nil || len(runner.calls) != 3 {
		t.Fatalf("expected exactly three attempts, got %d (%v)", len(runner.calls), err)
	}
}
