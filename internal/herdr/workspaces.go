package herdr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/security"
)

var (
	ErrWorkspaceNotFound  = errors.New("workspace_not_found")
	ErrWorkspaceAmbiguous = errors.New("ambiguous_workspace")
	ErrPathUnavailable    = errors.New("path_unavailable")
)

const (
	PathSourceWorktree = "worktree"
	PathSourcePaneCWD  = "pane_cwd"

	PathReasonNoPaneCWD           = "no_pane_cwd"
	PathReasonAmbiguousPaneCWD    = "ambiguous_pane_cwd"
	PathReasonPaneListUnavailable = "pane_list_unavailable"
	PathReasonInvalidWorktree     = "invalid_worktree_path"
)

type WorkspaceWorktree struct {
	CheckoutPath string `json:"checkout_path"`
}

type Workspace struct {
	WorkspaceID         string             `json:"workspace_id"`
	Label               string             `json:"label"`
	Focused             bool               `json:"focused"`
	Worktree            *WorkspaceWorktree `json:"worktree"`
	ResolvedProjectPath string             `json:"-"`
	PathSource          string             `json:"-"`
	PathReason          string             `json:"-"`
}

type Pane struct {
	WorkspaceID string `json:"workspace_id"`
	PaneID      string `json:"pane_id"`
	Focused     bool   `json:"focused"`
	CWD         string `json:"cwd"`
}

type WorkspaceView struct {
	WorkspaceID   string `json:"workspaceId"`
	Label         string `json:"label"`
	ProjectPath   string `json:"projectPath,omitempty"`
	Focused       bool   `json:"focused"`
	PathAvailable bool   `json:"pathAvailable"`
	PathSource    string `json:"pathSource,omitempty"`
	PathReason    string `json:"pathReason,omitempty"`
}

func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	data, err := c.Runner.Run(ctx, "workspace", "list")
	if err != nil {
		return nil, err
	}
	workspaces, err := ParseWorkspaces(data)
	if err != nil {
		return nil, err
	}
	if !needsPanePaths(workspaces) {
		return workspaces, nil
	}
	paneData, paneErr := c.Runner.Run(ctx, "pane", "list")
	if paneErr != nil {
		markPaneListUnavailable(workspaces)
		return workspaces, nil
	}
	panes, paneErr := ParsePanes(paneData)
	if paneErr != nil {
		markPaneListUnavailable(workspaces)
		return workspaces, nil
	}
	resolvePanePaths(workspaces, panes)
	return workspaces, nil
}

func ParseWorkspaces(data []byte) ([]Workspace, error) {
	var direct []Workspace
	if err := json.Unmarshal(data, &direct); err == nil {
		return direct, validateWorkspaces(direct)
	}
	var wrapper struct {
		Workspaces []Workspace     `json:"workspaces"`
		Result     json.RawMessage `json:"result"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("parse workspace list: %w", err)
	}
	if wrapper.Workspaces != nil {
		return wrapper.Workspaces, validateWorkspaces(wrapper.Workspaces)
	}
	if len(wrapper.Result) > 0 {
		var result struct {
			Workspaces []Workspace `json:"workspaces"`
		}
		if err := json.Unmarshal(wrapper.Result, &result); err == nil && result.Workspaces != nil {
			return result.Workspaces, validateWorkspaces(result.Workspaces)
		}
	}
	return nil, errors.New("workspace list did not contain workspaces")
}

func validateWorkspaces(workspaces []Workspace) error {
	for _, workspace := range workspaces {
		if workspace.WorkspaceID == "" {
			return errors.New("workspace entry is missing workspace_id")
		}
	}
	return nil
}

func ParsePanes(data []byte) ([]Pane, error) {
	var direct []Pane
	if err := json.Unmarshal(data, &direct); err == nil && direct != nil {
		return direct, validatePanes(direct)
	}
	var wrapper struct {
		Panes  []Pane          `json:"panes"`
		Result json.RawMessage `json:"result"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("parse pane list: %w", err)
	}
	if wrapper.Panes != nil {
		return wrapper.Panes, validatePanes(wrapper.Panes)
	}
	if len(wrapper.Result) > 0 {
		if err := json.Unmarshal(wrapper.Result, &direct); err == nil && direct != nil {
			return direct, validatePanes(direct)
		}
		var result struct {
			Panes []Pane `json:"panes"`
		}
		if err := json.Unmarshal(wrapper.Result, &result); err == nil && result.Panes != nil {
			return result.Panes, validatePanes(result.Panes)
		}
	}
	return nil, errors.New("pane list did not contain panes")
}

func validatePanes(panes []Pane) error {
	for _, pane := range panes {
		if pane.WorkspaceID == "" {
			return errors.New("pane entry is missing workspace_id")
		}
	}
	return nil
}

func needsPanePaths(workspaces []Workspace) bool {
	for _, workspace := range workspaces {
		if workspace.Worktree == nil || strings.TrimSpace(workspace.Worktree.CheckoutPath) == "" {
			return true
		}
	}
	return false
}

func markPaneListUnavailable(workspaces []Workspace) {
	for index := range workspaces {
		if workspaces[index].Worktree == nil || strings.TrimSpace(workspaces[index].Worktree.CheckoutPath) == "" {
			workspaces[index].PathReason = PathReasonPaneListUnavailable
		}
	}
}

func resolvePanePaths(workspaces []Workspace, panes []Pane) {
	byWorkspace := make(map[string][]Pane)
	for _, pane := range panes {
		byWorkspace[pane.WorkspaceID] = append(byWorkspace[pane.WorkspaceID], pane)
	}
	for index := range workspaces {
		workspace := &workspaces[index]
		if workspace.Worktree != nil && strings.TrimSpace(workspace.Worktree.CheckoutPath) != "" {
			continue
		}
		path, status := uniquePaneCWD(byWorkspace[workspace.WorkspaceID])
		if path != "" {
			workspace.ResolvedProjectPath = path
			workspace.PathSource = PathSourcePaneCWD
			workspace.PathReason = ""
			continue
		}
		workspace.PathReason = status
	}
}

func uniquePaneCWD(panes []Pane) (string, string) {
	paths := make(map[string]string)
	for _, pane := range panes {
		normalized, err := security.NormalizeWindowsPath(pane.CWD)
		if err != nil {
			continue
		}
		paths[strings.ToLower(normalized)] = normalized
	}
	if len(paths) == 0 {
		return "", PathReasonNoPaneCWD
	}
	if len(paths) > 1 {
		return "", PathReasonAmbiguousPaneCWD
	}
	for _, path := range paths {
		return path, ""
	}
	return "", PathReasonNoPaneCWD
}

func workspaceProjectPath(workspace Workspace) (string, string, string) {
	if workspace.Worktree != nil && strings.TrimSpace(workspace.Worktree.CheckoutPath) != "" {
		normalized, err := security.NormalizeWindowsPath(workspace.Worktree.CheckoutPath)
		if err != nil {
			return "", "", PathReasonInvalidWorktree
		}
		return normalized, PathSourceWorktree, ""
	}
	if workspace.ResolvedProjectPath != "" {
		normalized, err := security.NormalizeWindowsPath(workspace.ResolvedProjectPath)
		if err == nil {
			return normalized, workspace.PathSource, ""
		}
	}
	reason := workspace.PathReason
	if reason == "" {
		reason = PathReasonNoPaneCWD
	}
	return "", "", reason
}

func Views(workspaces []Workspace) []WorkspaceView {
	views := make([]WorkspaceView, 0, len(workspaces))
	for _, workspace := range workspaces {
		view := WorkspaceView{WorkspaceID: workspace.WorkspaceID, Label: workspace.Label, Focused: workspace.Focused}
		path, source, reason := workspaceProjectPath(workspace)
		if path != "" {
			view.ProjectPath = path
			view.PathAvailable = true
			view.PathSource = source
		} else {
			view.PathReason = reason
		}
		views = append(views, view)
	}
	return views
}

func ResolveWorkspace(workspaces []Workspace, projectPath, projectLabel string) (Workspace, error) {
	if _, err := security.NormalizeWindowsPath(projectPath); err != nil {
		return Workspace{}, ErrWorkspaceNotFound
	}
	matches := make([]Workspace, 0, 2)
	for _, workspace := range workspaces {
		workspacePath, _, _ := workspaceProjectPath(workspace)
		if workspacePath == "" {
			continue
		}
		if security.SameWindowsPath(workspacePath, projectPath) {
			matches = append(matches, workspace)
		}
	}
	if len(matches) == 0 {
		return Workspace{}, ErrWorkspaceNotFound
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	focused := filterWorkspaces(matches, func(workspace Workspace) bool { return workspace.Focused })
	if len(focused) == 1 {
		return focused[0], nil
	}
	labelled := filterWorkspaces(matches, func(workspace Workspace) bool {
		return strings.EqualFold(strings.TrimSpace(workspace.Label), strings.TrimSpace(projectLabel))
	})
	if len(labelled) == 1 {
		return labelled[0], nil
	}
	return Workspace{}, ErrWorkspaceAmbiguous
}

func filterWorkspaces(input []Workspace, keep func(Workspace) bool) []Workspace {
	output := make([]Workspace, 0, len(input))
	for _, workspace := range input {
		if keep(workspace) {
			output = append(output, workspace)
		}
	}
	return output
}
