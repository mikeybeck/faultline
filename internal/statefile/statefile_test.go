package statefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrependRecentDedupesAndCaps(t *testing.T) {
	var list []Project
	for i := 0; i < 10; i++ {
		list = prependRecent(list, Project{Dir: "/p", ConfigPath: filepath.Join("/p", "faultline.yaml") + string(rune('a'+i))})
	}
	if len(list) != recentCap {
		t.Fatalf("len = %d", len(list))
	}
	list = prependRecent(list, list[3])
	if list[0] != list[3] && list[0].ConfigPath != list[3].ConfigPath {
		// first should be the moved item; it should not appear twice
	}
	seen := map[string]int{}
	for _, p := range list {
		seen[p.ConfigPath]++
		if seen[p.ConfigPath] > 1 {
			t.Fatalf("duplicate %q", p.ConfigPath)
		}
	}
}

func TestRememberRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("LOCALAPPDATA", dir)
	cfg := filepath.Join(dir, "proj", "faultline.yaml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("sources:\n  - type: generic\n    path: a.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Remember(filepath.Dir(cfg), cfg); err != nil {
		t.Fatal(err)
	}
	got := RecentProjects()
	if len(got) != 1 || got[0].ConfigPath != cfg {
		t.Fatalf("recent = %+v", got)
	}
}
