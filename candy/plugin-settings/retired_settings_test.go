package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opencharly/sdk/kit"
	"github.com/opencharly/spec/hostenv"
)

// retired_settings_test.go — the retirement instrument for `engine.build`
// (opencharly/opencharly#481 leg 5). Each arm pins a property the retirement exists for: the key is
// answered BY NAME rather than as an unknown key, it is gone from the vocabulary, and a leftover
// entry in a user's config.yml is inert — measured, not assumed.

// withRetiredConfig points the settings surface at an isolated config.yml.
func withRetiredConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if body != "" {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
	}
	orig := hostenv.RuntimeConfigPath
	t.Cleanup(func() { hostenv.RuntimeConfigPath = orig })
	hostenv.RuntimeConfigPath = func() (string, error) { return path, nil }
	return path
}

// TestRetiredSettingAnswersByName is the whole point of the record: every verb answers the retired
// key as a RETIREMENT (naming what removed it and what did not), never as an unknown key — which
// would read as a typo and send the user hunting for a setting that is deliberately gone.
func TestRetiredSettingAnswersByName(t *testing.T) {
	withRetiredConfig(t, "")
	for _, arm := range []struct {
		name string
		call func() error
	}{
		// BOTH paths, because they are different code: GetConfigValue is the function the tests
		// called while the CLI's own entry point answered the generic unknown-key error — the arm
		// that would have caught it drives the SURFACE the command drives.
		{"get (function)", func() error { _, err := GetConfigValue(testCtx, nil, "engine.build"); return err }},
		{"get (CLI entry point)", func() error { _, err := resolveSettingsGet(testCtx, nil, "engine.build"); return err }},
		{"set", func() error { return SetConfigValue(testCtx, nil, "engine.build", "podman") }},
		{"reset", func() error { return ResetConfigValue(testCtx, nil, "engine.build") }},
	} {
		err := arm.call()
		if err == nil {
			t.Fatalf("%s: a retired key must be refused, got no error", arm.name)
		}
		msg := err.Error()
		for _, want := range []string{"engine.build was retired", "BuildKit", "no successor", "engine.run"} {
			if !strings.Contains(msg, want) {
				t.Fatalf("%s: the answer does not carry %q:\n%s", arm.name, want, msg)
			}
		}
		if strings.Contains(msg, "unknown config key") {
			t.Fatalf("%s: answered as an UNKNOWN key — the retirement was not named:\n%s", arm.name, msg)
		}
	}
}

// TestRetiredSettingIsNotInTheVocabulary pins the other half: the key is GONE from the listing and
// from the help text a user is shown, while the keys that did not retire stay.
func TestRetiredSettingIsNotInTheVocabulary(t *testing.T) {
	withRetiredConfig(t, "")
	for _, kv := range []string{"CHARLY_BUILD_ENGINE", "CHARLY_RUN_ENGINE", "CHARLY_RUN_MODE"} {
		t.Setenv(kv, "")
	}
	rows, err := ListConfigValues()
	if err != nil {
		t.Fatalf("ListConfigValues: %v", err)
	}
	var sawRun, sawBuild bool
	for _, r := range rows {
		switch r.Key {
		case "engine.run":
			sawRun = true
		case "engine.build":
			sawBuild = true
		}
	}
	if sawBuild {
		t.Fatalf("the listing still advertises the retired key: %+v", rows)
	}
	if !sawRun {
		t.Fatalf("the listing lost `engine.run` — the retirement took the wrong key: %+v", rows)
	}
	_, uerr := GetConfigValue(testCtx, nil, "definitely.not.a.key")
	if uerr == nil || !strings.Contains(uerr.Error(), "unknown config key") {
		t.Fatalf("a genuinely unknown key must still read as unknown, got %v", uerr)
	}
	// The UNKNOWN-key hint is where the vocabulary is advertised, so that is where the retired key
	// must be absent — the retired key itself answers as a retirement, never through this path.
	if strings.Contains(uerr.Error(), "engine.build") {
		t.Fatalf("the valid-key hint still offers the retired key:\n%s", uerr.Error())
	}
	if !strings.Contains(uerr.Error(), "valid: engine.run") {
		t.Fatalf("the hint lost the live engine key:\n%s", uerr.Error())
	}
}

// TestRetiredSettingLeftoverIsInert pins #481's D-1 measurement as a test rather than a note: a
// config.yml still carrying `engine.build` LOADS (the runtime config has no closed gate, unlike the
// project schema), the leftover is silently ignored, and every live key still works. That is why
// this retirement owes no migration-table entry — and the test is what keeps that true.
func TestRetiredSettingLeftoverIsInert(t *testing.T) {
	withRetiredConfig(t, "engine:\n  build: podman\n  run: docker\nrun_mode: quadlet\n")
	t.Setenv("CHARLY_RUN_ENGINE", "")

	cfg, err := kit.LoadRuntimeConfig()
	if err != nil {
		t.Fatalf("a leftover retired key must not fail the load (measured D-1): %v", err)
	}
	if cfg.Engine.Run != "docker" {
		t.Fatalf("the live sibling did not survive the leftover: %+v", cfg.Engine)
	}
	if got, err := GetConfigValue(testCtx, nil, "engine.run"); err != nil || got != "docker" {
		t.Fatalf("engine.run = (%q, %v), want (\"docker\", nil)", got, err)
	}
	if _, err := GetConfigValue(testCtx, nil, "engine.build"); err == nil {
		t.Fatal("the retired key must still be answered as retired even when the file carries it")
	}
}

// TestRetiredSettingsCarryAnAnswer keeps the record honest for the NEXT retirement: a row without a
// reason or without an explicit successor statement is not a record, it is a deletion.
func TestRetiredSettingsCarryAnAnswer(t *testing.T) {
	rows := retiredSettings()
	if len(rows) == 0 {
		t.Fatal("the record is empty — a retirement with no row is a silent disappearance")
	}
	for _, r := range rows {
		if r.Key == "" || r.Why == "" || r.Instead == "" {
			t.Fatalf("retired row %+v must carry key, why, and what replaces it", r)
		}
	}
}
