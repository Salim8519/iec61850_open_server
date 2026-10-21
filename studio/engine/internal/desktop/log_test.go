package desktop

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/testutil"
)

func TestLogFileFormat(t *testing.T) {
	data := testutil.TempDir(t)
	stamp := time.Date(2026, 10, 20, 15, 4, 5, 123456789, time.Local)
	log := openLogFile(data, func() time.Time { return stamp })
	log.Warning(appWindowLogger, "App window was handed to another browser process")
	log.Error("studio_app.launcher.entrypoint", "ScadaBench could not start: %s", "port")
	log.Close()
	log.Info("x", "after close is discarded")
	got, err := os.ReadFile(filepath.Join(data, "logs", "launcher-20261020.log"))
	if err != nil {
		t.Fatal(err)
	}
	want := "2026-10-20 15:04:05,123 WARNING studio_app.launcher.app_window: App window was handed to another browser process" + lineEnd +
		"2026-10-20 15:04:05,123 ERROR studio_app.launcher.entrypoint: ScadaBench could not start: port" + lineEnd
	if string(got) != want {
		t.Fatalf("log file\n got %q\nwant %q", got, want)
	}
}

func TestLogTerminalFormatAndNil(t *testing.T) {
	var out bytes.Buffer
	log := &Logger{out: &out, now: time.Now}
	log.Info("studio_app.launcher.entrypoint", "port %d", 8765)
	if got := out.String(); got != "INFO:studio_app.launcher.entrypoint:port 8765"+lineEnd {
		t.Fatalf("terminal line %q", got)
	}
	var none *Logger
	none.Warning("x", "nil logger discards")
	none.Close()
}

func TestLogUncreatableFolderDiscards(t *testing.T) {
	data := testutil.TempDir(t)
	file := filepath.Join(data, "data")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	log := openLogFile(file, time.Now)
	log.Info("x", "discarded")
	log.Close()
}
