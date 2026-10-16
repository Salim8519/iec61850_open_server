// Package sales keeps the owner's permanent payment record: one JSON line per paid
// order, refund or reversal, appended and synced to disk, never rewritten.
package sales

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Record is one money event. Event is "paid", "refunded" or "reversed".
type Record struct {
	Time       int64  `json:"time"`
	Event      string `json:"event"`
	OrderID    string `json:"order_id"`
	CaptureID  string `json:"capture_id,omitempty"`
	Amount     string `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
	Email      string `json:"email,omitempty"`
	Name       string `json:"name,omitempty"`
	Country    string `json:"country,omitempty"`
	PayerID    string `json:"payer_id,omitempty"`
	LicenseKey string `json:"license_key,omitempty"`
	HWID       string `json:"hwid,omitempty"`
	Mode       string `json:"mode,omitempty"`
}

// Book appends records to <data>/sales.jsonl. Payments are rare and each line is
// under 1 KB, so the file stays tiny; it is the owner's record, not a log.
type Book struct {
	path string
	mu   sync.Mutex
}

// Path is the record file inside the data folder.
func Path(dataDir string) string { return filepath.Join(dataDir, "sales.jsonl") }

// Open returns the book for dataDir; the file is created on the first record.
func Open(dataDir string) *Book { return &Book{path: Path(dataDir)} }

// Append writes one record and syncs it before returning.
func (b *Book) Append(record Record) error {
	if b == nil {
		return errors.New("sales book is not open")
	}
	line, err := json.Marshal(record)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	file, err := os.OpenFile(b.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(append(line, '\n')); err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Read returns every record in order; a missing file means no sales yet.
func Read(dataDir string) ([]Record, error) {
	file, err := os.Open(Path(dataDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var records []Record
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		var record Record
		if json.Unmarshal(scanner.Bytes(), &record) == nil {
			records = append(records, record)
		}
	}
	return records, scanner.Err()
}
