package chrome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateProfileName(t *testing.T) {
	for _, ok := range []string{"wb-ms", "oz-cr", "wb_hb", "a", "x1"} {
		if err := ValidateProfileName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "WB-MS", "../x", "wb ms", "wb/ms", `wb\ms`, "-x", strings.Repeat("a", 65)} {
		if err := ValidateProfileName(bad); err == nil {
			t.Errorf("%q принято", bad)
		}
	}
}

func TestProfileDir(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir, err := ProfileDir("wb-ms")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "wb-ms" || filepath.Base(filepath.Dir(dir)) != "chrome-profiles" {
		t.Errorf("каталог профиля: %s", dir)
	}
	if _, err := ProfileDir("../evil"); err == nil {
		t.Error("выход из каталога профилей принят")
	}
}

func TestReadPortFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "DevToolsActivePort")
	if _, ok := readPortFile(p); ok {
		t.Error("нет файла — не должно быть адреса")
	}
	if err := os.WriteFile(p, []byte("9222\n/devtools/browser/abc-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if u, ok := readPortFile(p); !ok || u != "ws://127.0.0.1:9222/devtools/browser/abc-1" {
		t.Errorf("адрес: %q %v", u, ok)
	}
	// Файл записан не полностью.
	if err := os.WriteFile(p, []byte("9222\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := readPortFile(p); ok {
		t.Error("неполный файл принят")
	}
}

func TestFindExecutableFromEnv(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "chrome")
	if err := os.WriteFile(exe, nil, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvChromePath, exe)
	if got, err := FindExecutable(); err != nil || got != exe {
		t.Errorf("FindExecutable: %q %v", got, err)
	}
	t.Setenv(EnvChromePath, filepath.Join(t.TempDir(), "nope"))
	if _, err := FindExecutable(); err == nil {
		t.Error("несуществующий путь принят")
	}
}
