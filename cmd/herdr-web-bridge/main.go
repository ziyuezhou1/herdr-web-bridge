package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ziyuezhou1/herdr-web-bridge/internal/bindings"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/bridge"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/buildinfo"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/config"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/herdr"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/ipc"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/native"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/platform"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/protocol"
	"github.com/ziyuezhou1/herdr-web-bridge/internal/security"
)

type check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, security.RedactLog(err.Error()))
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && strings.HasPrefix(args[0], "chrome-extension://") {
		return runNative(args[0])
	}
	if len(args) == 0 {
		return usageError()
	}
	switch args[0] {
	case "version", "--version", "-version":
		writeJSON(map[string]string{"name": buildinfo.Name, "version": buildinfo.Version})
		return nil
	case "native-host":
		origin := "chrome-extension://" + buildinfo.ExtensionID + "/"
		if len(args) > 1 {
			origin = args[1]
		}
		return runNative(origin)
	case "broker":
		return runBroker()
	case "doctor":
		return runDoctor()
	case "list-bindings":
		return listBindings()
	case "list-workspaces":
		return listWorkspaces()
	case "open":
		return openBinding(args[1:])
	case "notify-test":
		return notifyTest()
	case "install-status":
		return installStatus()
	default:
		return usageError()
	}
}

func newService(writer *native.Writer) (*bridge.Service, error) {
	path, err := bindings.DefaultPath()
	if err != nil {
		return nil, err
	}
	executable, err := bridge.ExecutablePath()
	if err != nil {
		return nil, err
	}
	herdrPath, err := platform.HerdrExecutable()
	if err != nil {
		return nil, err
	}
	service := bridge.New(bindings.NewStore(path), herdr.NewClient(herdr.ExecRunner{Path: herdrPath}), executable)
	installConfig, configErr := config.Load()
	if configErr != nil {
		return nil, configErr
	}
	service.AllowedExtensionID = installConfig.ExtensionID
	service.Writer = writer
	return service, nil
}

func runNative(origin string) error {
	writer := native.NewWriter(os.Stdout)
	service, err := newService(writer)
	if err != nil {
		return err
	}
	if !service.ValidateOrigin(origin) {
		return errors.New("native host rejected an unexpected extension origin")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if server, listenErr := ipc.Listen(); listenErr == nil {
		go func() {
			if serveErr := server.Serve(ctx, service.HandleIPC); serveErr != nil {
				fmt.Fprintln(os.Stderr, "IPC:", security.RedactLog(serveErr.Error()))
			}
		}()
	} else {
		fmt.Fprintln(os.Stderr, "IPC unavailable:", security.RedactLog(listenErr.Error()))
	}
	host := native.Host{Input: os.Stdin, Writer: writer, Handler: service.HandleNative, Log: os.Stderr}
	return host.Serve()
}

func runBroker() error {
	service, err := newService(nil)
	if err != nil {
		return err
	}
	server, err := ipc.Listen()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return server.Serve(ctx, service.HandleIPC)
}

func runDoctor() error {
	checks := make([]check, 0, 9)
	herdrPath, err := platform.HerdrExecutable()
	checks = append(checks, makeCheck("herdr_cli", err == nil, valueOrError(herdrPath, err)))
	client := herdr.NewClient(herdr.ExecRunner{Path: herdrPath})
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	statusErr := client.Status(ctx)
	checks = append(checks, makeCheck("herdr_running", statusErr == nil, errorText(statusErr)))
	workspaces, workspaceErr := client.ListWorkspaces(ctx)
	checks = append(checks, makeCheck("workspace_list", workspaceErr == nil, countOrError("workspaces", len(workspaces), workspaceErr)))
	notifyErr := client.Notify(ctx, "Herdr Web Bridge 测试", "doctor 已成功连接 Herdr 通知接口", "none")
	checks = append(checks, makeCheck("herdr_notification", notifyErr == nil, errorText(notifyErr)))
	plus, plusErr := client.HasHerdrPlus(ctx)
	plusOK := plusErr == nil && plus
	plusDetails := errorText(plusErr)
	if plusErr == nil && !plus {
		plusDetails = "not installed; optional Quick Actions are unavailable; install manually with: herdr plugin install cloudmanic/herdr-plus"
	} else if plusErr != nil {
		plusDetails = plusErr.Error() + "; if absent, install manually with: herdr plugin install cloudmanic/herdr-plus"
	}
	checks = append(checks, makeCheck("herdr_plus", plusOK, plusDetails))
	registration := platform.NativeHostRegistration()
	checks = append(checks, makeCheck("native_manifest", registration.Registered, registration.Details))
	ping, pingErr := ipc.Call("ping", map[string]interface{}{}, 800*time.Millisecond)
	checks = append(checks, makeCheck("local_ipc", pingErr == nil && ping.OK, errorText(pingErr)))
	connection, connectionErr := ipc.Call("status", map[string]interface{}{}, 800*time.Millisecond)
	extensionConnected := connectionErr == nil && connection.OK && responseBool(connection, "extensionConnected")
	checks = append(checks, makeCheck("edge_extension", extensionConnected, errorText(connectionErr)))
	storePath, pathErr := bindings.DefaultPath()
	var bindingErr error
	bindingCount := 0
	if pathErr == nil {
		file, loadErr := bindings.NewStore(storePath).Load()
		bindingErr = loadErr
		bindingCount = len(file.Bindings)
	} else {
		bindingErr = pathErr
	}
	checks = append(checks, makeCheck("bindings_config", bindingErr == nil, countOrError("bindings", bindingCount, bindingErr)))
	overall := true
	for _, item := range checks {
		if !item.OK && item.Name != "herdr_plus" {
			overall = false
		}
	}
	writeJSON(map[string]interface{}{"version": buildinfo.Version, "ok": overall, "checks": checks, "privacy": "browser content, query strings, credentials, cookies, and tokens are omitted"})
	if !overall {
		return errors.New("doctor found one or more required checks that need attention")
	}
	return nil
}

func listBindings() error {
	path, err := bindings.DefaultPath()
	if err != nil {
		return err
	}
	file, err := bindings.NewStore(path).Load()
	if err != nil {
		return err
	}
	items := make([]map[string]interface{}, 0, len(file.Bindings))
	for _, binding := range file.Bindings {
		items = append(items, map[string]interface{}{
			"id": binding.ID, "projectPath": binding.ProjectPath, "projectLabel": binding.ProjectLabel,
			"url": security.SafeURLForLog(binding.URL), "adapter": binding.Adapter,
			"lastState": binding.LastState, "syncPending": binding.SyncPending,
		})
	}
	writeJSON(map[string]interface{}{"schemaVersion": file.SchemaVersion, "bindings": items})
	return nil
}

func listWorkspaces() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	herdrPath, err := platform.HerdrExecutable()
	if err != nil {
		return err
	}
	client := herdr.NewClient(herdr.ExecRunner{Path: herdrPath})
	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		return err
	}
	writeJSON(map[string]interface{}{"workspaces": herdr.Views(workspaces)})
	return nil
}

func openBinding(args []string) error {
	flags := flag.NewFlagSet("open", flag.ContinueOnError)
	bindingID := flags.String("binding", "", "trusted binding UUID")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *bindingID == "" {
		return errors.New("open requires --binding <binding_id>")
	}
	response, ipcErr := ipc.Call("open_binding", protocol.BindingIDPayload{BindingID: *bindingID}, 1500*time.Millisecond)
	if ipcErr == nil && response.OK {
		writeJSON(response.Result)
		return nil
	}
	path, err := bindings.DefaultPath()
	if err != nil {
		return err
	}
	binding, err := bindings.NewStore(path).Get(*bindingID)
	if err != nil {
		return err
	}
	if _, err := security.ValidateURL(binding.URL, true); err != nil {
		return err
	}
	if err := platform.OpenURL(binding.URL); err != nil {
		return err
	}
	writeJSON(map[string]interface{}{"status": "fallback_opened", "reason": "extension_or_ipc_unavailable", "bindingId": binding.ID})
	return nil
}

func notifyTest() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	herdrPath, err := platform.HerdrExecutable()
	if err != nil {
		return err
	}
	client := herdr.NewClient(herdr.ExecRunner{Path: herdrPath})
	if err := client.Notify(ctx, "Herdr Web Bridge 测试", "本地桥接通知测试成功", "done"); err != nil {
		return err
	}
	writeJSON(map[string]string{"status": "notification_sent"})
	return nil
}

func installStatus() error {
	registration := platform.NativeHostRegistration()
	installedPath := platform.InstalledExecutable()
	installConfig, configErr := config.Load()
	configuredExtensionID := buildinfo.ExtensionID
	if configErr == nil {
		configuredExtensionID = installConfig.ExtensionID
		if installConfig.ExecutablePath != "" {
			installedPath = installConfig.ExecutablePath
		}
	}
	_, fileErr := os.Stat(installedPath)
	writeJSON(map[string]interface{}{
		"version":             buildinfo.Version,
		"installedExecutable": installedPath,
		"executablePresent":   fileErr == nil,
		"nativeHost":          registration,
		"expectedExtensionId": configuredExtensionID,
		"installConfigValid":  configErr == nil,
	})
	return nil
}

func writeJSON(value interface{}) {
	encoded, _ := json.MarshalIndent(value, "", "  ")
	fmt.Println(string(encoded))
}

func makeCheck(name string, ok bool, details string) check {
	status := "failed"
	if ok {
		status = "passed"
	}
	return check{Name: name, OK: ok, Status: status, Details: security.TruncateRunes(security.RedactLog(details), 200)}
}

func errorText(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}

func valueOrError(value string, err error) string {
	if err != nil {
		return err.Error()
	}
	return value
}

func countOrError(label string, count int, err error) string {
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%d %s", count, label)
}

func responseBool(response protocol.Response, key string) bool {
	data, err := json.Marshal(response.Result)
	if err != nil {
		return false
	}
	var result map[string]interface{}
	if json.Unmarshal(data, &result) != nil {
		return false
	}
	value, _ := result[key].(bool)
	return value
}

func usageError() error {
	name := filepath.Base(os.Args[0])
	return fmt.Errorf("usage: %s <doctor|list-bindings|list-workspaces|open|notify-test|install-status>", name)
}
