package extdist

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Install copies the bundled extension into the user config dir and returns that path.
func Install(src fs.FS) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("extension: %w", err)
	}
	dest := filepath.Join(base, "faultline", "extension")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", fmt.Errorf("extension: %w", err)
	}
	root := "extension"
	err = fs.WalkDir(src, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		out := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		in, err := src.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(f, in)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		return "", fmt.Errorf("extension: %w", err)
	}
	return dest, nil
}

// Zip packs dir into a zip whose entries live under faultline-extension/.
func Zip(dir string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == ".DS_Store" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(filepath.Join("faultline-extension", rel)))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		_ = f.Close()
		return copyErr
	})
	if err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func looksLikeExtension(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "manifest.json"))
	return err == nil && !st.IsDir()
}

// Prefer a repo checkout's extension/ folder when present.
func PreferLocal(bundled string) string {
	wd, err := os.Getwd()
	if err != nil {
		return bundled
	}
	local := filepath.Join(wd, "extension")
	if looksLikeExtension(local) {
		return local
	}
	if looksLikeExtension(bundled) {
		return bundled
	}
	return bundled
}

func JoinURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "http://127.0.0.1:9477/extension.zip"
	}
	if !strings.Contains(addr, "://") {
		addr = "http://" + addr
	}
	return strings.TrimRight(addr, "/") + "/extension.zip"
}
