// Команда mp-reviews-probe — пробник кабинетов (SPEC §11, этап 0).
//
// Запускает видимый Chrome с постоянным профилем кабинета и записывает, какие запросы отправляет
// кабинет, пока пользователь сам работает в нём. Сам пробник в кабинете ничего не делает.
// Результат — один zip в каталоге -out. Порядок работы — в README.md рядом.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/3177423-prog/mp-reviews/internal/cdp"
	"github.com/3177423-prog/mp-reviews/internal/chrome"
	"github.com/3177423-prog/mp-reviews/internal/probe"
)

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("mp-reviews-probe", flag.ContinueOnError)
	profile := fs.String("profile", "", "профиль кабинета, например wb-ms или oz-cr (обязательно)")
	startURL := fs.String("url", "", "адрес, который открыть при запуске (например https://seller.wildberries.ru/)")
	outDir := fs.String("out", ".", "каталог для zip с результатом")
	chromePath := fs.String("chrome", "", "путь к chrome.exe (по умолчанию — "+chrome.EnvChromePath+" или стандартная установка)")
	verbose := fs.Bool("v", false, "подробный журнал в stderr")
	fs.Usage = func() {
		_, _ = fmt.Fprintln(fs.Output(), "Использование: mp-reviews-probe -profile wb-ms [-url адрес] [-out каталог] [-chrome путь]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if *profile == "" {
		fs.Usage()
		return 2
	}
	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	if err := probeRun(*profile, *startURL, *outDir, *chromePath, log); err != nil {
		log.Error("пробник завершился с ошибкой", "err", err)
		return 1
	}
	return 0
}

func probeRun(profile, startURL, outDir, chromePath string, log *slog.Logger) error {
	profileDir, err := chrome.ProfileDir(profile)
	if err != nil {
		return err
	}
	if chromePath == "" {
		if chromePath, err = chrome.FindExecutable(); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return fmt.Errorf("каталог результата: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Printf("Профиль %s: %s\nЗапускаю Chrome: %s\n", profile, profileDir, chromePath)
	browser, err := chrome.Launch(ctx, chrome.Options{ExecPath: chromePath, UserDataDir: profileDir})
	if err != nil {
		return err
	}
	defer browser.Wait(10 * time.Second)

	dialCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	conn, err := cdp.Dial(dialCtx, browser.WebSocketURL)
	cancel()
	if err != nil {
		browser.Kill()
		return err
	}

	rec, err := probe.New(conn, probe.Options{Profile: profile, OutDir: outDir, Logger: log})
	if err != nil {
		_ = conn.Close()
		browser.Kill()
		return err
	}
	startCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	err = rec.Start(startCtx)
	if err == nil && startURL != "" {
		err = rec.Navigate(startCtx, startURL)
	}
	cancel()
	if err != nil {
		// Без подписки на события запись бесполезна: сохраняем то, что есть, и выходим.
		zipPath, finErr := rec.Finish()
		_ = conn.Close()
		browser.Kill()
		if finErr == nil {
			fmt.Println("Частичная запись:", zipPath)
		}
		return fmt.Errorf("не удалось начать запись: %w", err)
	}

	fmt.Printf("\nЗапись идёт (рабочий каталог %s).\n", rec.WorkDir())
	fmt.Println("Работайте в кабинете в открывшемся окне Chrome как обычно — пробник только наблюдает.")
	fmt.Println("Чтобы отметить шаг: введите его название и нажмите Enter (сохранятся снимок страницы и скриншот).")
	fmt.Println("Чтобы завершить: q и Enter (или закройте Chrome).")

	lines := make(chan string)
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()

loop:
	for {
		fmt.Print("шаг> ")
		select {
		case <-ctx.Done():
			fmt.Println("\nПрервано, сохраняю запись.")
			break loop
		case <-rec.Done():
			fmt.Println("\nChrome закрыт, сохраняю запись.")
			break loop
		case line, ok := <-lines:
			if !ok {
				break loop
			}
			line = strings.TrimSpace(line)
			switch line {
			case "":
				continue
			case "q", "Q":
				break loop
			}
			markCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			res := rec.Mark(markCtx, line)
			cancel()
			fmt.Printf("  отметка %02d сохранена", res.N)
			if res.HTML != "" || res.PNG != "" {
				fmt.Printf(": %s %s", res.HTML, res.PNG)
			}
			fmt.Println()
			for _, e := range res.Errors {
				fmt.Println("  не получилось:", e)
			}
		}
	}

	zipPath, finErr := rec.Finish()

	// Закрываем Chrome штатно, чтобы профиль (и вход в кабинет) сохранился.
	select {
	case <-conn.Done():
	default:
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = conn.Call(closeCtx, "", "Browser.close", nil)
		cancel()
		_ = conn.Close()
	}

	if finErr != nil {
		return fmt.Errorf("%w (данные остались в %s)", finErr, rec.WorkDir())
	}
	fmt.Printf("\nГотово: %s\nФайл содержит тела запросов кабинета — никуда его не публикуйте.\n", zipPath)
	if err := conn.Err(); err != nil && !errors.Is(err, cdp.ErrClosed) {
		log.Warn("соединение с Chrome", "err", err)
	}
	return nil
}
