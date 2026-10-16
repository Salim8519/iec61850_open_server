package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"substation-license-server/internal/api"
	"substation-license-server/internal/config"
	"substation-license-server/internal/sales"
)

// payments prints the permanent payment record (newest last); --json for tools.
func payments(args []string) error {
	var asJSON *bool
	dataDir, err := flags("payments", args, func(set *flag.FlagSet) { asJSON = set.Bool("json", false, "print JSON lines") })
	if err != nil {
		return err
	}
	records, err := sales.Read(dataDir)
	if err != nil {
		return err
	}
	if *asJSON {
		encoder := json.NewEncoder(os.Stdout)
		for _, record := range records {
			encoder.Encode(record)
		}
		return nil
	}
	if len(records) == 0 {
		fmt.Println("No payments recorded yet.")
		return nil
	}
	table := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(table, "TIME (UTC)\tEVENT\tAMOUNT\tBUYER\tEMAIL\tCOUNTRY\tKEY\tPAYPAL ORDER\tMODE")
	totals := map[string]float64{}
	for _, r := range records {
		fmt.Fprintf(table, "%s\t%s\t%s %s\t%s\t%s\t%s\t%s\t%s\t%s\n", time.Unix(r.Time, 0).UTC().Format("2006-01-02 15:04"),
			r.Event, r.Amount, r.Currency, dash(r.Name), dash(r.Email), dash(r.Country), r.LicenseKey, r.OrderID, r.Mode)
		var value float64
		fmt.Sscan(r.Amount, &value)
		if r.Mode == "live" && r.Event == "paid" {
			totals[r.Currency] += value
		} else if r.Mode == "live" {
			totals[r.Currency] -= value
		}
	}
	table.Flush()
	for currency, total := range totals {
		fmt.Printf("Live total after refunds: %.2f %s\n", total, currency)
	}
	return nil
}

func dash(text string) string {
	if text == "" {
		return "-"
	}
	return text
}

// printUsage shows how much disk the service uses against the 100 MB budget.
func printUsage(cfg config.Config) {
	mb := func(bytes int64) string { return fmt.Sprintf("%.1f MB", float64(bytes)/(1<<20)) }
	data := api.FolderBytes(cfg.DataDir)
	programs := fileBytes("/opt/substation-license/license-server") + fileBytes("/usr/local/bin/license-server-admin") +
		fileBytes("/usr/local/bin/cloudflared-scadabench")
	records, _ := sales.Read(cfg.DataDir)
	fmt.Println("Payments:       ", len(records), "recorded (license-server payments)")
	fmt.Println("Disk (data):    ", mb(data), "of", mb(cfg.DataBudgetBytes()), "budget; logs capped at",
		mb(int64(cfg.LogMaxMegabytes*cfg.LogMaxFiles)<<20))
	fmt.Println("Disk (total):   ", mb(data+programs), "including programs (target under 100 MB)")
}

func fileBytes(path string) int64 {
	info, err := os.Lstat(filepath.Clean(path))
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return 0
	}
	return info.Size()
}
