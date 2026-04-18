// Command loadmodules loads kernel modules at boot.
//
// Module paths are passed as command-line arguments, relative to
// /lib/modules/<kernel-release>/. They are loaded in order, so
// dependencies must come before the modules that need them.
//
// Configure the module list via CommandLineFlags in gokrazy's
// config.json. See the README for examples.
//
// loadmodules exits with status 125 on success, which tells gokrazy
// not to supervise (restart) the process.
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

func main() {
	modules := os.Args[1:]
	if len(modules) == 0 {
		fmt.Fprintf(os.Stderr, "loadmodules: no modules specified, nothing to do\n")
		os.Exit(125)
	}

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
