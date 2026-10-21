package main

import (
	"fmt"
	"io"
	"os"

	"substation-license-server/internal/release"
)

// publishRelease saves the newest app version from a JSON record on standard input
// (written by website/deploy.cmd). Restart the service afterwards so it signs the new answer.
func publishRelease(args []string) error {
	dataDir, err := flags("release", args, nil)
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 16<<10))
	if err != nil {
		return err
	}
	info, err := release.Parse(raw)
	if err != nil {
		return err
	}
	if err := release.Save(dataDir, info); err != nil {
		return err
	}
	fmt.Println("Saved release", info.Version, "from", info.Date, "| installer", info.Installer.URL)
	return nil
}
