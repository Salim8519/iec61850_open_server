package launcher

import (
	"os"
	"path/filepath"

	"studio-engine/internal/pyjson"
)

// RecordName is the launcher record in the data folder.
const RecordName = "launcher.json"

// WriteRecord is entrypoint.write_launcher_record: json.dumps({"url": url} plus
// "window_pid" when set), written as UTF-8 without a final newline.
func WriteRecord(directory, url string, windowPID int) error {
	record := pyjson.NewObject().Set("url", url)
	if windowPID != 0 {
		record.Set("window_pid", int64(windowPID))
	}
	data, err := pyjson.Dumps(record)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, RecordName), data, 0o644)
}

// ReadWindowPID is the second launch's json.loads(launcher.json).get("window_pid"): 0 when
// the file is missing, unreadable, not an object or has no integer window_pid.
func ReadWindowPID(directory string) int {
	data, err := os.ReadFile(filepath.Join(directory, RecordName))
	if err != nil {
		return 0
	}
	decoded, err := pyjson.Decode(data)
	if err != nil {
		return 0
	}
	record, isObject := decoded.(*pyjson.Object)
	if !isObject {
		return 0
	}
	value, _ := record.Get("window_pid")
	pid, _ := value.(int64)
	return int(pid)
}
