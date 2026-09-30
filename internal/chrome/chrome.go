// Package chrome запускает установленный Chrome с постоянным профилем кабинета и открытым
// портом DevTools. Используется пробником (этап 0) и потом браузерным агентом (этап 3):
// каталог профиля у них общий, чтобы вход в кабинет выполнялся один раз.
package chrome

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// EnvChromePath — переменная окружения с путём к chrome.exe / chromium, если автопоиск не находит.
const EnvChromePath = "MP_REVIEWS_CHROME_PATH"

var profileNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// ValidateProfileName проверяет имя профиля (например, wb-ms): только строчные латинские буквы,
// цифры, '-' и '_' — имя становится именем каталога.
func ValidateProfileName(name string) error {
	if !profileNameRe.MatchString(name) {
		return fmt.Errorf("имя профиля %q: допустимы строчные латинские буквы, цифры, '-' и '_', до 64 символов", name)
	}
	return nil
}

// ProfilesRoot — корень каталогов профилей: %LOCALAPPDATA%\mp-reviews\chrome-profiles на Windows,
// ~/.cache/mp-reviews/chrome-profiles (или $XDG_CACHE_HOME) на Linux.
func ProfilesRoot() (string, error) {
	base := ""
	if runtime.GOOS == "windows" {
		base = os.Getenv("LOCALAPPDATA")
	}
	if base == "" {
		d, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("каталог профилей: %w", err)
		}
		base = d
	}
	return filepath.Join(base, "mp-reviews", "chrome-profiles"), nil
}

// ProfileDir — каталог профиля Chrome для кабинета name.
func ProfileDir(name string) (string, error) {
	if err := ValidateProfileName(name); err != nil {
		return "", err
	}
	root, err := ProfilesRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

// FindExecutable ищет Chrome: сначала переменная MP_REVIEWS_CHROME_PATH, потом стандартные места.
func FindExecutable() (string, error) {
	if p := os.Getenv(EnvChromePath); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("%s=%q: %w", EnvChromePath, p, err)
		}
		return p, nil
	}
	var candidates []string
	if runtime.GOOS == "windows" {
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
			if base := os.Getenv(env); base != "" {
				candidates = append(candidates, filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"))
			}
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("chrome не найден: укажите путь флагом -chrome или переменной %s", EnvChromePath)
}

// Options — параметры запуска.
type Options struct {
	ExecPath    string
	UserDataDir string
	// Headless — только для тестов: пробник и агент у пользователя работают с видимым окном.
	Headless  bool
	ExtraArgs []string
	// Output — куда писать stdout/stderr Chrome (nil — отбросить).
	Output io.Writer
	// StartTimeout — сколько ждать появления порта DevTools (по умолчанию 30 с).
	StartTimeout time.Duration
}

// Browser — запущенный процесс Chrome.
type Browser struct {
	// WebSocketURL — адрес DevTools уровня браузера.
	WebSocketURL string

	cmd  *exec.Cmd
	done chan struct{}
	err  error
}

// ErrProfileBusy — Chrome с этим профилем уже открыт: новый процесс передал управление ему и вышел.
var ErrProfileBusy = errors.New("chrome с этим профилем уже запущен — закройте все его окна и повторите")

// Launch запускает Chrome с --remote-debugging-port=0 и ждёт, пока он запишет выбранный порт
// в файл DevToolsActivePort каталога профиля. Порт слушается только на 127.0.0.1.
func Launch(ctx context.Context, opts Options) (*Browser, error) {
	if opts.ExecPath == "" || opts.UserDataDir == "" {
		return nil, errors.New("chrome: не заданы путь к Chrome или каталог профиля")
	}
	if err := os.MkdirAll(opts.UserDataDir, 0o700); err != nil {
		return nil, fmt.Errorf("chrome: каталог профиля: %w", err)
	}
	portFile := filepath.Join(opts.UserDataDir, "DevToolsActivePort")
	// Старый файл от прошлого запуска дал бы неверный порт.
	if err := os.Remove(portFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("chrome: %w", err)
	}

	args := []string{
		"--user-data-dir=" + opts.UserDataDir,
		"--remote-debugging-port=0",
		"--remote-debugging-address=127.0.0.1",
		"--no-first-run",
		"--no-default-browser-check",
	}
	if opts.Headless {
		args = append(args, "--headless=new")
	}
	args = append(args, opts.ExtraArgs...)
	args = append(args, "about:blank")

	cmd := exec.Command(opts.ExecPath, args...)
	cmd.Stdout = opts.Output
	cmd.Stderr = opts.Output
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("chrome: запуск %s: %w", opts.ExecPath, err)
	}
	b := &Browser{cmd: cmd, done: make(chan struct{})}
	go func() {
		b.err = cmd.Wait()
		close(b.done)
	}()

	timeout := opts.StartTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		if u, ok := readPortFile(portFile); ok {
			b.WebSocketURL = u
			return b, nil
		}
		select {
		case <-b.done:
			if b.err == nil {
				return nil, ErrProfileBusy
			}
			return nil, fmt.Errorf("chrome: процесс завершился при старте: %w", b.err)
		case <-deadline.C:
			b.Kill()
			return nil, errors.New("chrome: не дождались порта DevTools")
		case <-ctx.Done():
			b.Kill()
			return nil, ctx.Err()
		case <-tick.C:
		}
	}
}

// readPortFile читает DevToolsActivePort: первая строка — порт, вторая — путь websocket браузера.
func readPortFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return "", false
	}
	port, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil || port <= 0 || port > 65535 {
		return "", false
	}
	p := strings.TrimSpace(lines[1])
	if !strings.HasPrefix(p, "/devtools/browser/") {
		return "", false
	}
	return fmt.Sprintf("ws://127.0.0.1:%d%s", port, p), true
}

// Done закрывается, когда процесс Chrome завершился.
func (b *Browser) Done() <-chan struct{} { return b.done }

// Wait ждёт завершения процесса не дольше timeout; по истечении убивает его.
func (b *Browser) Wait(timeout time.Duration) {
	select {
	case <-b.done:
	case <-time.After(timeout):
		b.Kill()
	}
}

// Kill завершает процесс Chrome.
func (b *Browser) Kill() {
	if b.cmd.Process != nil {
		_ = b.cmd.Process.Kill()
	}
	<-b.done
}
