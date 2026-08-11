package lootmeta

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestRefreshDownloadsMasterlistAndPrelude(t *testing.T) {
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		switch r.URL.Path {
		case "/fallout4/v0.29/masterlist.yaml":
			_, _ = w.Write([]byte("plugins: []\n"))
		case "/prelude/v0.29/prelude.yaml":
			_, _ = w.Write([]byte("globals: []\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	service := Service{DataDir: dir, BaseURL: server.URL, HTTPClient: server.Client()}
	status, err := service.Refresh(context.Background(), sdk.PluginActivationSpec{
		LOOTGameID:           "fallout4vr",
		LOOTMasterlistGameID: "fallout4",
		LOOTPrelude:          true,
	})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if status.GameID != "fallout4vr" || status.MasterlistGameID != "fallout4" {
		t.Fatalf("status IDs = %q/%q", status.GameID, status.MasterlistGameID)
	}
	if !status.Masterlist.Exists || !status.Prelude.Exists {
		t.Fatalf("expected cached files, got masterlist=%+v prelude=%+v", status.Masterlist, status.Prelude)
	}
	if !contains(requested, "/fallout4/v0.29/masterlist.yaml") || !contains(requested, "/prelude/v0.29/prelude.yaml") {
		t.Fatalf("requested paths = %+v", requested)
	}
	masterlistBody, err := os.ReadFile(filepath.Join(dir, "loot", "fallout4vr", "masterlist", "masterlist.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(masterlistBody) != "plugins: []\n" {
		t.Fatalf("masterlist body = %q", string(masterlistBody))
	}
}

func TestStatusRejectsUnsafeIDs(t *testing.T) {
	_, err := (Service{DataDir: t.TempDir()}).Status(sdk.PluginActivationSpec{LOOTGameID: "../fallout4"})
	if err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("Status() error = %v", err)
	}
}

func TestUserlistReadWriteNormalizesVortexShape(t *testing.T) {
	dir := t.TempDir()
	service := Service{DataDir: dir}
	spec := sdk.PluginActivationSpec{LOOTGameID: "fallout4"}

	written, err := service.WriteUserlist(spec, Userlist{
		Plugins: []UserlistPlugin{
			{Name: " Example.esp ", After: []string{"A.esm"}, Requires: []string{"Base.esm"}},
			{Name: "example.ESP", Group: "Late Loaders", Incompatible: []string{"Old.esp"}},
		},
		Groups: []UserlistGroup{
			{Name: "Late Loaders", After: []string{"default", "DEFAULT"}},
		},
	})
	if err != nil {
		t.Fatalf("WriteUserlist() error = %v", err)
	}
	if got := written.Summary(); got.Plugins != 1 || got.Rules != 3 || got.Groups != 1 || got.GroupRules != 1 {
		t.Fatalf("summary = %+v", got)
	}

	read, err := service.ReadUserlist(spec)
	if err != nil {
		t.Fatalf("ReadUserlist() error = %v", err)
	}
	if len(read.Plugins) != 1 {
		t.Fatalf("plugins = %+v", read.Plugins)
	}
	plugin := read.Plugins[0]
	if plugin.Name != "Example.esp" || plugin.Group != "Late Loaders" {
		t.Fatalf("plugin = %+v", plugin)
	}
	if len(plugin.After) != 1 || plugin.After[0] != "A.esm" {
		t.Fatalf("after = %+v", plugin.After)
	}
	if len(plugin.Requires) != 1 || plugin.Requires[0] != "Base.esm" {
		t.Fatalf("requires = %+v", plugin.Requires)
	}
	if len(plugin.Incompatible) != 1 || plugin.Incompatible[0] != "Old.esp" {
		t.Fatalf("incompatible = %+v", plugin.Incompatible)
	}

	body, err := os.ReadFile(filepath.Join(dir, "loot", "fallout4", "userlist.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"plugins:", "after:", "req:", "inc:", "groups:"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("userlist.yaml missing %q:\n%s", want, string(body))
		}
	}
}

func TestWriteUserlistRejectsDuplicateVortexRule(t *testing.T) {
	dir := t.TempDir()
	service := Service{DataDir: dir}
	spec := sdk.PluginActivationSpec{LOOTGameID: "fallout4"}

	_, err := service.WriteUserlist(spec, Userlist{
		Plugins: []UserlistPlugin{
			{Name: "Example.esp", After: []string{"Fallout4.esm"}},
			{Name: " example.ESP ", Requires: []string{"Other.esp"}, After: []string{" fallout4.ESM "}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), `duplicate LOOT userlist rule "example.ESP after fallout4.ESM"`) {
		t.Fatalf("WriteUserlist() duplicate error = %v", err)
	}
}

func TestStatusWarnsForMissingUserlistGroups(t *testing.T) {
	dir := t.TempDir()
	service := Service{DataDir: dir}
	spec := sdk.PluginActivationSpec{LOOTGameID: "fallout4"}
	if err := os.MkdirAll(filepath.Join(dir, "loot", "fallout4", "masterlist"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "loot", "fallout4", "masterlist", "masterlist.yaml"), []byte("groups:\n  - name: Early\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.WriteUserlist(spec, Userlist{
		Plugins: []UserlistPlugin{{Name: "Example.esp", Group: "Missing Plugin Group"}},
		Groups:  []UserlistGroup{{Name: "Late", After: []string{"Early", "Missing After Group"}}},
	}); err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status.UserlistWarning, "Missing After Group") || !strings.Contains(status.UserlistWarning, "Missing Plugin Group") {
		t.Fatalf("warning = %q", status.UserlistWarning)
	}
}

func TestPluginRulesForProfileReadsMasterlistAndUserlistRules(t *testing.T) {
	dir := t.TempDir()
	service := Service{DataDir: dir}
	spec := sdk.PluginActivationSpec{LOOTGameID: "fallout4"}
	masterlistDir := filepath.Join(dir, "loot", "fallout4", "masterlist")
	if err := os.MkdirAll(masterlistDir, 0o700); err != nil {
		t.Fatal(err)
	}
	masterlist := `plugins:
  - name: Main.esp
    req:
      - Fallout4.esm
      - name: DLCUltraHighResolution.esm
        display: High Resolution DLC
    inc:
      - OldPatch.esp
`
	if err := os.WriteFile(filepath.Join(masterlistDir, "masterlist.yaml"), []byte(masterlist), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.WriteUserlistForProfile(spec, 7, Userlist{
		Plugins: []UserlistPlugin{{Name: "Main.esp", Requires: []string{"Loose/File.txt"}, Incompatible: []string{"Conflict.esp"}}},
	}); err != nil {
		t.Fatal(err)
	}
	rules, err := service.PluginRulesForProfile(spec, 7)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, rule := range rules {
		got[rule.Plugin+"\x00"+rule.Kind+"\x00"+rule.Target] = rule.Display
	}
	for key, display := range map[string]string{
		"Main.esp\x00requires\x00Fallout4.esm":               "Fallout4.esm",
		"Main.esp\x00requires\x00DLCUltraHighResolution.esm": "High Resolution DLC",
		"Main.esp\x00incompatible\x00OldPatch.esp":           "OldPatch.esp",
		"Main.esp\x00requires\x00Loose/File.txt":             "Loose/File.txt",
		"Main.esp\x00incompatible\x00Conflict.esp":           "Conflict.esp",
	} {
		if got[key] != display {
			t.Fatalf("rule %q display = %q, want %q; rules=%+v", key, got[key], display, rules)
		}
	}
}

func TestProfileUserlistsAreIsolated(t *testing.T) {
	dir := t.TempDir()
	service := Service{DataDir: dir}
	spec := sdk.PluginActivationSpec{LOOTGameID: "fallout4"}

	if _, err := service.WriteUserlistForProfile(spec, 10, Userlist{Plugins: []UserlistPlugin{{Name: "A.esp", After: []string{"Fallout4.esm"}}}}); err != nil {
		t.Fatalf("WriteUserlistForProfile(10) error = %v", err)
	}
	if _, err := service.WriteUserlistForProfile(spec, 11, Userlist{Plugins: []UserlistPlugin{{Name: "B.esp", Requires: []string{"A.esp"}}}}); err != nil {
		t.Fatalf("WriteUserlistForProfile(11) error = %v", err)
	}

	left, err := service.ReadUserlistForProfile(spec, 10)
	if err != nil {
		t.Fatal(err)
	}
	right, err := service.ReadUserlistForProfile(spec, 11)
	if err != nil {
		t.Fatal(err)
	}
	if len(left.Plugins) != 1 || left.Plugins[0].Name != "A.esp" {
		t.Fatalf("left userlist = %+v", left)
	}
	if len(right.Plugins) != 1 || right.Plugins[0].Name != "B.esp" {
		t.Fatalf("right userlist = %+v", right)
	}

	status, err := service.StatusForProfile(spec, 10)
	if err != nil {
		t.Fatal(err)
	}
	if status.ProfileID != 10 || status.UserlistRules.Rules != 1 || !strings.Contains(status.Userlist.Path, filepath.Join("profiles", "10", "userlist.yaml")) {
		t.Fatalf("status = %+v", status)
	}
}

func TestCopyUserlistForProfileSeedsNewProfile(t *testing.T) {
	dir := t.TempDir()
	service := Service{DataDir: dir}
	spec := sdk.PluginActivationSpec{LOOTGameID: "fallout4"}

	if _, err := service.WriteUserlistForProfile(spec, 10, Userlist{
		Plugins: []UserlistPlugin{{Name: "Example.esp", Group: "Late"}},
		Groups:  []UserlistGroup{{Name: "Late", After: []string{"default"}}},
	}); err != nil {
		t.Fatal(err)
	}
	copied, err := service.CopyUserlistForProfile(spec, 10, 12)
	if err != nil {
		t.Fatalf("CopyUserlistForProfile() error = %v", err)
	}
	if !copied {
		t.Fatal("expected userlist copy")
	}
	read, err := service.ReadUserlistForProfile(spec, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Plugins) != 1 || read.Plugins[0].Group != "Late" || len(read.Groups) != 1 {
		t.Fatalf("copied userlist = %+v", read)
	}
}

func TestSorterStatusReportsHelperAvailability(t *testing.T) {
	service := Service{DataDir: t.TempDir(), SorterCommand: filepath.Join(t.TempDir(), "missing-sorter")}
	status := service.SorterStatus()
	if status.Available || status.Status != "blocked" || !strings.Contains(status.Message, "dmm-loot-sorter") {
		t.Fatalf("missing sorter status = %+v", status)
	}

	helper := fakeSorter(t, `{"sorted_plugins":["B.esp","A.esp"],"engine":"fake-libloot"}`)
	service.SorterCommand = helper
	status = service.SorterStatus()
	if !status.Available || status.Status != "ready" || status.Command != helper {
		t.Fatalf("available sorter status = %+v", status)
	}
}

func TestSortInvokesHelperWithProfileLOOTPaths(t *testing.T) {
	dir := t.TempDir()
	gamePath := filepath.Join(dir, "game")
	if err := os.MkdirAll(gamePath, 0o700); err != nil {
		t.Fatal(err)
	}
	aPath := filepath.Join(gamePath, "A.esp")
	bPath := filepath.Join(gamePath, "B.esp")
	for _, path := range []string{aPath, bPath} {
		if err := os.WriteFile(path, []byte("plugin"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "loot", "fallout4", "masterlist"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "loot", "fallout4", "masterlist", "masterlist.yaml"), []byte("plugins: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(dir, "request.json")
	helper := fakeSorterWithCapture(t, capture, `{"sorted_plugins":["B.esp","A.esp"],"warnings":["sorted"],"engine":"fake-libloot"}`)
	service := Service{DataDir: dir, SorterCommand: helper}

	out, err := service.Sort(context.Background(), sdk.PluginActivationSpec{LOOTGameID: "fallout4"}, 42, SortInput{
		GamePath: gamePath,
		Plugins: []SortPlugin{
			{Name: "A.esp", Path: aPath, Source: "dmm", Active: true},
			{Name: "B.esp", Path: bPath, Source: "dmm", Active: true},
		},
		CurrentOrder: []string{"A.esp", "B.esp"},
	})
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	if strings.Join(out.SortedPlugins, ",") != "B.esp,A.esp" || out.Engine != "fake-libloot" {
		t.Fatalf("sort output = %+v", out)
	}
	body, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	var req sorterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatal(err)
	}
	if req.GameID != "fallout4" || !strings.Contains(req.Userlist, filepath.Join("profiles", "42", "userlist.yaml")) {
		t.Fatalf("sort request paths = %+v", req)
	}
	if req.Contract != "dmm-loot-sorter.v1" || len(req.Plugins) != 2 {
		t.Fatalf("sort request = %+v", req)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func fakeSorter(t *testing.T, response string) string {
	t.Helper()
	return fakeSorterWithCapture(t, "", response)
}

func fakeSorterWithCapture(t *testing.T, capturePath, response string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dmm-loot-sorter")
	var script string
	if capturePath == "" {
		script = "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '" + strings.ReplaceAll(response, "'", "'\\''") + "'\n"
	} else {
		script = "#!/bin/sh\ncat > '" + strings.ReplaceAll(capturePath, "'", "'\\''") + "'\nprintf '%s\\n' '" + strings.ReplaceAll(response, "'", "'\\''") + "'\n"
	}
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
