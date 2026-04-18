// Command loadmodules loads kernel modules for the BPI-R1 at boot.
//
// It loads the rtl8xxxu WiFi driver module. The WiFi stack dependencies
// (cfg80211, mac80211, libarc4) are built into the kernel.
//
// This program is intended to be added to a gokrazy instance as a package.
// It loads the modules and exits — gokrazy should not supervise it.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func release() string {
	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		fmt.Fprintf(os.Stderr, "loadmodules: uname: %v\n", err)
		os.Exit(1)
	}
	return string(uts.Release[:bytes.IndexByte(uts.Release[:], 0)])
}

func loadModule(rel, mod string) error {
	p := filepath.Join("/lib/modules", rel, mod)
	f, err := os.Open(p)
	if err != nil {
		return fmt.Errorf("open %s: %v", p, err)
	}
	defer f.Close()
	if err := unix.FinitModule(int(f.Fd()), "", 0); err != nil {
		if err == unix.EEXIST || err == unix.EBUSY {
			fmt.Fprintf(os.Stderr, "loadmodules: %s: already loaded\n", mod)
			return nil
		}
		return fmt.Errorf("FinitModule(%s): %v", mod, err)
	}
	fmt.Fprintf(os.Stderr, "loadmodules: %s: loaded\n", mod)
	return nil
}

var modules = []string{
	"kernel/drivers/net/wireless/realtek/rtl8xxxu/rtl8xxxu.ko",
}

func main() {
	rel := release()
	var failed bool
	for _, mod := range modules {
		if err := loadModule(rel, mod); err != nil {
			fmt.Fprintf(os.Stderr, "loadmodules: %v\n", err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
	// Exit 125 tells gokrazy "don't supervise": do not restart this program.
	os.Exit(125)
}
