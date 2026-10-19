package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"substation-license-server/internal/adminauth"
	"substation-license-server/internal/audit"
)

// adminSetup creates or replaces the owner page login (ADR-0072). The password is
// read from standard input only, never from the command line, so it cannot show up
// in shell history or the process list. Replacing the login signs everyone out.
func adminSetup(args []string) error {
	var asJSON *bool
	dataDir, err := flags("admin-setup", args, func(set *flag.FlagSet) {
		set.Bool("password-stdin", true, "read the new password from standard input (always on)")
		asJSON = set.Bool("json", false, "print the result as JSON for the Windows helper")
	})
	if err != nil {
		return err
	}
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return errors.New("type the new password on standard input")
	}
	password := strings.TrimRight(strings.TrimPrefix(line, string(rune(0xFEFF))), "\r\n")
	keeper := adminauth.NewKeeper(dataDir)
	previous, _, err := keeper.Load()
	if err != nil {
		// A damaged file: start from a time-based epoch so no old session survives.
		previous = adminauth.Account{Epoch: time.Now().Unix()}
	}
	account, enrollment, err := adminauth.NewAccount(password, previous.Epoch, time.Now())
	if err != nil {
		return err
	}
	if err := keeper.Replace(account); err != nil {
		return err
	}
	audit.Open(dataDir).Add("admin-setup", "server console", "new password, authenticator and recovery codes", true, time.Now())
	if *asJSON {
		raw, _ := json.Marshal(enrollment)
		fmt.Println(string(raw))
		return nil
	}
	fmt.Println("Owner login saved. Every browser that was signed in is now signed out.")
	fmt.Println()
	fmt.Println("1. In Google Authenticator (or any authenticator app) choose 'Enter a setup key'.")
	fmt.Println("   Account name: ScadaBench owner   Key:", enrollment.SecretGrouped, "  Type: Time based")
	fmt.Println("2. Write these recovery codes on paper. Each one works once if you lose your phone:")
	for i, code := range enrollment.RecoveryCodes {
		fmt.Printf("   %2d. %s\n", i+1, code)
	}
	return nil
}

// adminStatus says whether the owner login exists, without showing any secret.
func adminStatus(args []string) error {
	dataDir, err := flags("admin-status", args, nil)
	if err != nil {
		return err
	}
	account, ready, err := adminauth.NewKeeper(dataDir).Load()
	if err != nil {
		return err
	}
	if !ready {
		fmt.Println("Owner login: not set up (run admin-setup.cmd on your PC)")
		return nil
	}
	fmt.Printf("Owner login: set up %s, %d recovery codes left\n",
		time.Unix(account.CreatedAt, 0).UTC().Format("2006-01-02 15:04 UTC"), len(account.RecoveryHashes))
	return nil
}
