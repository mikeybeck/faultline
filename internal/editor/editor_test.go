package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildVSCode(t *testing.T) {
	o := Opener{Command: "code"}
	bin, args, err := o.build("/tmp/PaymentController.php", 81)
	if err != nil {
		t.Fatal(err)
	}
	if bin != "code" {
		t.Fatalf("bin = %q", bin)
	}
	if len(args) != 2 || args[0] != "--goto" || args[1] != "/tmp/PaymentController.php:81" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildPhpStorm(t *testing.T) {
	o := Opener{Command: "phpstorm"}
	bin, args, err := o.build("/tmp/User.php", 41)
	if err != nil {
		t.Fatal(err)
	}
	_ = bin
	if len(args) < 3 || args[0] != "--line" || args[1] != "41" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildPhpStormAbsolutePath(t *testing.T) {
	path := `C:\Program Files\JetBrains\PhpStorm 2025.1\bin\phpstorm64.exe`
	o := Opener{Command: path}
	bin, args, err := o.build(`C:\proj\User.php`, 41)
	if err != nil {
		t.Fatal(err)
	}
	if bin != path {
		t.Fatalf("bin = %q", bin)
	}
	if len(args) != 3 || args[0] != "--line" || args[1] != "41" || args[2] != `C:\proj\User.php` {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildQuotedWindowsPath(t *testing.T) {
	path := `C:\Program Files\JetBrains\PhpStorm 2025.1\bin\phpstorm64.exe`
	o := Opener{Command: `"` + path + `"`}
	bin, args, err := o.build(`C:\proj\User.php`, 12)
	if err != nil {
		t.Fatal(err)
	}
	if bin != path {
		t.Fatalf("bin = %q", bin)
	}
	if len(args) != 3 || args[0] != "--line" || args[1] != "12" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildTemplate(t *testing.T) {
	o := Opener{Command: "nvim +{line} {file}"}
	bin, args, err := o.build("/tmp/a.php", 10)
	if err != nil {
		t.Fatal(err)
	}
	if bin != "nvim" {
		t.Fatalf("bin = %q", bin)
	}
	if len(args) != 2 || args[0] != "+10" || args[1] != "/tmp/a.php" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildTemplateQuotedPath(t *testing.T) {
	o := Opener{Command: `"C:\Program Files\phpstorm64.exe" --line {line} {file}`}
	bin, args, err := o.build(`C:\app\a.php`, 8)
	if err != nil {
		t.Fatal(err)
	}
	if bin != `C:\Program Files\phpstorm64.exe` {
		t.Fatalf("bin = %q", bin)
	}
	if len(args) != 3 || args[0] != "--line" || args[1] != "8" || args[2] != `C:\app\a.php` {
		t.Fatalf("args = %#v", args)
	}
}

func TestNewestGlobPicksLatestVersion(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "PhpStorm 2023.1", "bin")
	newer := filepath.Join(dir, "PhpStorm 2025.1", "bin")
	if err := os.MkdirAll(older, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newer, 0o755); err != nil {
		t.Fatal(err)
	}
	olderExe := filepath.Join(older, "phpstorm64.exe")
	newerExe := filepath.Join(newer, "phpstorm64.exe")
	if err := os.WriteFile(olderExe, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newerExe, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	got := newestGlob(filepath.Join(dir, "PhpStorm*", "bin", "phpstorm64.exe"))
	if got != newerExe {
		t.Fatalf("got %q want %q", got, newerExe)
	}
}

func TestFlavorOf(t *testing.T) {
	cases := map[string]string{
		`C:\x\phpstorm64.exe`: "phpstorm",
		"phpstorm.cmd":        "phpstorm",
		"code.cmd":            "vscode",
		"Cursor.exe":          "vscode",
		"nvim":                "generic",
	}
	for in, want := range cases {
		if got := flavorOf(in); got != want {
			t.Errorf("flavorOf(%q) = %q, want %q", in, got, want)
		}
	}
}
