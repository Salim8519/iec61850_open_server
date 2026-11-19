package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// restore unpacks a backup made by backup into the data folder. It writes only the
// names a backup carries and never overwrites a file, so a live data folder is safe.
func restore(args []string) error {
	var from *string
	dataDir, err := flags("restore", args, func(set *flag.FlagSet) {
		from = set.String("from", "", "backup file made by the backup command")
	})
	if err != nil {
		return err
	}
	if *from == "" {
		return errors.New("--from is required")
	}
	file, err := os.Open(*from)
	if err != nil {
		return err
	}
	defer file.Close()
	zipped, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("not a backup file: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	archive := tar.NewReader(zipped)
	var restored []string
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if header.Name != "licenses.db" && !slices.Contains(backupFiles, header.Name) {
			return fmt.Errorf("unexpected file %q in the backup", header.Name)
		}
		if err := restoreFile(filepath.Join(dataDir, header.Name), archive); err != nil {
			return err
		}
		restored = append(restored, header.Name)
	}
	fmt.Println("Restored into", dataDir+":", restored)
	return nil
}

func restoreFile(path string, content io.Reader) error {
	target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%s already exists; restore only into an empty data folder", path)
	}
	if err != nil {
		return err
	}
	if _, err := io.Copy(target, content); err != nil {
		target.Close()
		return err
	}
	return target.Close()
}
