package settings

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencharly/sdk/kit"
	"github.com/opencharly/spec/container"
	"github.com/opencharly/spec/hostenv"
)

// config_test.go — ported from charly/runtime_config_test.go (wave γ, the config-subsystem
// relocation into this plugin). Every test here exercises a non-credential-touching key
// (engine.*, run_mode, auto_enable, bind_address), so ctx/exec are safely nil/background —
// GetConfigValue/SetConfigValue/ResetConfigValue never reach credentialCall for these keys.
// Coverage preserved 1:1 from the core tests.

var testCtx = context.Background()

func TestSetConfigValue_Validates(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	// engine.run's vocabulary is spec's #EngineName (podman | docker | containerd),
	// emitted as spec.EngineNames, plus the `auto` resolution selector. BOTH arms are
	// pinned: `containerd` is ACCEPTED and the retired `nerdctl` word is REJECTED.
	// Checking acceptance alone would pass on the pre-cutover tree under the old
	// spelling, so the pair is what makes this test able to fail without the change.
	if err := SetConfigValue(testCtx, nil, "engine.run", "containerd"); err != nil {
		t.Errorf("engine.run must accept containerd (spec #EngineName): %v", err)
	}
	if val, err := GetConfigValue(testCtx, nil, "engine.run"); err != nil || val != "containerd" {
		t.Errorf("after set, engine.run = (%q, %v), want (\"containerd\", nil)", val, err)
	}

	err := SetConfigValue(testCtx, nil, "engine.run", "nerdctl")
	if err == nil {
		t.Fatal("engine.run must reject the retired nerdctl word — it is the CLI binary, not an engine name")
	}
	// The rejection names the CURRENT vocabulary: `auto` plus every member of
	// spec.EngineNames, read from the contract module rather than transcribed here.
	for _, want := range append([]string{"auto"}, container.EngineNames()...) {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the rejection must name %q (current vocabulary):\n%s", want, err)
		}
	}
	// ...and must not ADVERTISE the retired word. Its one permitted occurrence is
	// the rejected value itself (`got "nerdctl"`); a second occurrence means the
	// offered list still carries it.
	if got := strings.Count(err.Error(), "nerdctl"); got != 1 {
		t.Errorf("nerdctl must appear only as the rejected value, got %d occurrences:\n%s", got, err)
	}

	err = SetConfigValue(testCtx, nil, "run_mode", "swarm")
	if err == nil {
		t.Error("expected error for invalid run_mode value")
	}

	err = SetConfigValue(testCtx, nil, "engine.run", "podman")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := GetConfigValue(testCtx, nil, "engine.run")
	if err != nil {
		t.Fatalf("GetConfigValue() error: %v", err)
	}
	if val != "podman" {
		t.Errorf("GetConfigValue() = %q, want %q", val, "podman")
	}
}

func TestResetConfigValue(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	if err := SetConfigValue(testCtx, nil, "engine.run", "podman"); err != nil {
		t.Fatal(err)
	}
	if err := ResetConfigValue(testCtx, nil, "engine.run"); err != nil {
		t.Fatal(err)
	}

	val, _ := GetConfigValue(testCtx, nil, "engine.run")
	if val != "" {
		t.Errorf("after reset, GetConfigValue() = %q, want empty", val)
	}
}

func TestResetConfigValue_All(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	if err := SetConfigValue(testCtx, nil, "engine.run", "podman"); err != nil {
		t.Fatal(err)
	}
	if err := SetConfigValue(testCtx, nil, "run_mode", "quadlet"); err != nil {
		t.Fatal(err)
	}
	if err := ResetConfigValue(testCtx, nil, ""); err != nil {
		t.Fatal(err)
	}

	cfg, _ := kit.LoadRuntimeConfig()
	if cfg.Engine.Run != "" || cfg.RunMode != "" {
		t.Errorf("after full reset, config should be empty, got %+v", cfg)
	}
}

func TestListConfigValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	_ = os.Unsetenv("CHARLY_RUN_ENGINE")
	_ = os.Unsetenv("CHARLY_RUN_MODE")
	_ = os.Unsetenv("CHARLY_AUTO_ENABLE")
	_ = os.Unsetenv("CHARLY_BIND_ADDRESS")
	_ = os.Unsetenv("CHARLY_ENCRYPTED_STORAGE_PATH")
	_ = os.Unsetenv("CHARLY_SECRET_BACKEND")
	_ = os.Unsetenv("CHARLY_VM_BACKEND")
	_ = os.Unsetenv("CHARLY_VM_DISK_SIZE")
	_ = os.Unsetenv("CHARLY_VM_RAM")
	_ = os.Unsetenv("CHARLY_VM_CPUS")

	if err := SetConfigValue(testCtx, nil, "engine.run", "podman"); err != nil {
		t.Fatal(err)
	}

	vals, err := ListConfigValues()
	if err != nil {
		t.Fatalf("ListConfigValues() error: %v", err)
	}
	// 19 rows: the retired `engine.build` is no longer among them (its own arms pin that).
	if len(vals) != 19 {
		t.Fatalf("expected 19 values, got %d", len(vals))
	}

	// engine.run is the FIRST row now, and it comes from config
	if vals[0].Key != "engine.run" || vals[0].Value != "podman" || vals[0].Source != "config" {
		t.Errorf("engine.run entry: %+v", vals[0])
	}
	// engine.rootful should be default "auto"
	if vals[1].Key != "engine.rootful" || vals[1].Value != "auto" || vals[1].Source != "default" {
		t.Errorf("engine.rootful entry: %+v", vals[1])
	}
	// auto_enable should be default true
	if vals[3].Key != "auto_enable" || vals[3].Value != "true" || vals[3].Source != "default" {
		t.Errorf("auto_enable entry: %+v", vals[3])
	}
	// bind_address should be default 127.0.0.1
	if vals[4].Key != "bind_address" || vals[4].Value != "127.0.0.1" || vals[4].Source != "default" {
		t.Errorf("bind_address entry: %+v", vals[4])
	}
}

func TestGetConfigValue_UnknownKey(t *testing.T) {
	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) {
		return filepath.Join(t.TempDir(), "config.yml"), nil
	}

	_, err := GetConfigValue(testCtx, nil, "foo.bar")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

// assertConfigKeySetGetReset points hostenv.RuntimeConfigPath at a fresh temp config and
// exercises one config key's set/get/invalid/reset lifecycle: set valid1 + read
// it back, set valid2 + read it back, reject invalid, then reset to empty.
// Shared by the per-key *_SetGetReset tests (R3).
func assertConfigKeySetGetReset(t *testing.T, key, valid1, valid2, invalid string) {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	t.Cleanup(func() { hostenv.RuntimeConfigPath = orig })
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	// First valid value.
	if err := SetConfigValue(testCtx, nil, key, valid1); err != nil {
		t.Fatalf("SetConfigValue(%s, %s) error: %v", key, valid1, err)
	}
	val, err := GetConfigValue(testCtx, nil, key)
	if err != nil {
		t.Fatalf("GetConfigValue(%s) error: %v", key, err)
	}
	if val != valid1 {
		t.Errorf("GetConfigValue(%s) = %q, want %q", key, val, valid1)
	}

	// Second valid value.
	if err := SetConfigValue(testCtx, nil, key, valid2); err != nil {
		t.Fatalf("SetConfigValue(%s, %s) error: %v", key, valid2, err)
	}
	val, _ = GetConfigValue(testCtx, nil, key)
	if val != valid2 {
		t.Errorf("GetConfigValue(%s) = %q, want %q", key, val, valid2)
	}

	// Invalid value is rejected.
	if err := SetConfigValue(testCtx, nil, key, invalid); err == nil {
		t.Errorf("expected error for invalid %s value", key)
	}

	// Reset clears it.
	if err := ResetConfigValue(testCtx, nil, key); err != nil {
		t.Fatalf("ResetConfigValue(%s) error: %v", key, err)
	}
	val, _ = GetConfigValue(testCtx, nil, key)
	if val != "" {
		t.Errorf("after reset, GetConfigValue(%s) = %q, want empty", key, val)
	}
}

func TestAutoEnable_SetGetReset(t *testing.T) {
	assertConfigKeySetGetReset(t, "auto_enable", "true", "false", "yes")
}

func TestAutoEnable_EnvOverridesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	_ = os.Unsetenv("CHARLY_RUN_ENGINE")
	_ = os.Unsetenv("CHARLY_RUN_MODE")
	_ = os.Unsetenv("CHARLY_BIND_ADDRESS")

	// Config says false
	if err := SetConfigValue(testCtx, nil, "auto_enable", "false"); err != nil {
		t.Fatal(err)
	}

	// Env says true
	_ = os.Setenv("CHARLY_AUTO_ENABLE", "true")
	defer os.Unsetenv("CHARLY_AUTO_ENABLE") //nolint:errcheck

	rt, err := kit.ResolveRuntime()
	if err != nil {
		t.Fatalf("ResolveRuntime() error: %v", err)
	}
	if !rt.AutoEnable {
		t.Error("AutoEnable should be true when CHARLY_AUTO_ENABLE=true overrides config")
	}
}

func TestAutoEnable_ListConfigValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	_ = os.Unsetenv("CHARLY_RUN_ENGINE")
	_ = os.Unsetenv("CHARLY_RUN_MODE")
	_ = os.Unsetenv("CHARLY_AUTO_ENABLE")
	_ = os.Unsetenv("CHARLY_BIND_ADDRESS")

	if err := SetConfigValue(testCtx, nil, "auto_enable", "true"); err != nil {
		t.Fatal(err)
	}

	vals, err := ListConfigValues()
	if err != nil {
		t.Fatalf("ListConfigValues() error: %v", err)
	}

	// Find auto_enable entry
	found := false
	for _, v := range vals {
		if v.Key == "auto_enable" {
			found = true
			if v.Value != "true" || v.Source != "config" {
				t.Errorf("auto_enable entry: %+v, want value=true source=config", v)
			}
		}
	}
	if !found {
		t.Error("auto_enable not found in ListConfigValues output")
	}
}

func TestBindAddress_SetGetReset(t *testing.T) {
	assertConfigKeySetGetReset(t, "bind_address", "0.0.0.0", "127.0.0.1", "192.168.1.1")
}

func TestBindAddress_EnvOverridesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	_ = os.Unsetenv("CHARLY_RUN_ENGINE")
	_ = os.Unsetenv("CHARLY_RUN_MODE")
	_ = os.Unsetenv("CHARLY_AUTO_ENABLE")

	// Config says 127.0.0.1
	if err := SetConfigValue(testCtx, nil, "bind_address", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}

	// Env says 0.0.0.0
	_ = os.Setenv("CHARLY_BIND_ADDRESS", "0.0.0.0")
	defer os.Unsetenv("CHARLY_BIND_ADDRESS") //nolint:errcheck

	rt, err := kit.ResolveRuntime()
	if err != nil {
		t.Fatalf("ResolveRuntime() error: %v", err)
	}
	if rt.BindAddress != "0.0.0.0" {
		t.Errorf("BindAddress = %q, want %q (env should override config)", rt.BindAddress, "0.0.0.0")
	}
}

// TestVmImageDir_SetGetReset asserts the configurable VM image root: a valid
// path round-trips through set/get, an empty value is rejected, and reset
// clears it. The resolver itself (default "image", env precedence) is covered by
// TestVmDiskRoot_Configurable in the sdk module.
func TestVmImageDir_SetGetReset(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	if err := SetConfigValue(testCtx, nil, "vm.image_dir", "/srv/vm-images"); err != nil {
		t.Fatalf("set vm.image_dir: %v", err)
	}
	val, err := GetConfigValue(testCtx, nil, "vm.image_dir")
	if err != nil || val != "/srv/vm-images" {
		t.Fatalf("get vm.image_dir = %q err=%v, want /srv/vm-images", val, err)
	}

	if err := SetConfigValue(testCtx, nil, "vm.image_dir", ""); err == nil {
		t.Error("expected error for an empty vm.image_dir")
	}

	if err := ResetConfigValue(testCtx, nil, "vm.image_dir"); err != nil {
		t.Fatal(err)
	}
	val, _ = GetConfigValue(testCtx, nil, "vm.image_dir")
	if val != "" {
		t.Errorf("after reset, vm.image_dir = %q, want empty", val)
	}
}

// TestVmImageDir_ListAndEnv covers the surfaces the set/get/reset test does not: the
// `settings list` entry (key present, default "image") and (c) the CHARLY_VM_IMAGE_DIR
// env override winning over EITHER the default OR a config value. The resolver itself is
// covered in the sdk module; this pins THIS plugin's list/surface behavior.
func TestVmImageDir_ListAndEnv(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	orig := hostenv.RuntimeConfigPath
	defer func() { hostenv.RuntimeConfigPath = orig }()
	hostenv.RuntimeConfigPath = func() (string, error) { return configPath, nil }

	// (a) list carries vm.image_dir with the default "image".
	vals, err := ListConfigValues()
	if err != nil {
		t.Fatalf("ListConfigValues: %v", err)
	}
	found := false
	for _, v := range vals {
		if v.Key == "vm.image_dir" {
			found = true
			if v.Value != "image" {
				t.Errorf("vm.image_dir default in list = %q, want image", v.Value)
			}
		}
	}
	if !found {
		t.Error("vm.image_dir missing from ListConfigValues")
	}

	// (b) an authored config value is reflected in list.
	if err := SetConfigValue(testCtx, nil, "vm.image_dir", "/srv/x"); err != nil {
		t.Fatalf("set: %v", err)
	}
	vals, _ = ListConfigValues()
	for _, v := range vals {
		if v.Key == "vm.image_dir" && v.Value != "/srv/x" {
			t.Errorf("vm.image_dir list after set = %q, want /srv/x", v.Value)
		}
	}

	// (c) the env override WINS over the config value in the list.
	t.Setenv("CHARLY_VM_IMAGE_DIR", "/srv/from-env")
	vals, _ = ListConfigValues()
	for _, v := range vals {
		if v.Key == "vm.image_dir" && v.Value != "/srv/from-env" {
			t.Errorf("vm.image_dir list with env override = %q, want /srv/from-env", v.Value)
		}
	}
}
