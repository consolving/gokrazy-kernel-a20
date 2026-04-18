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
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func openConsole() *os.File {
	f, err := os.OpenFile("/dev/console", os.O_WRONLY, 0)
	if err != nil {
		return os.Stderr
	}
	return f
}

func release() string {
	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		log.Fatalf("uname: %v", err)
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
			log.Printf("%s: already loaded", mod)
			return nil
		}
		return fmt.Errorf("finit_module(%s): %v", mod, err)
	}
	log.Printf("%s: loaded", mod)
	return nil
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("loadmodules: ")
	log.SetOutput(openConsole())

	modules := os.Args[1:]
	if len(modules) == 0 {
		log.Print("no modules specified, nothing to do")
		os.Exit(125)
	}

	rel := release()
	var failed bool
	for _, mod := range modules {
		if err := loadModule(rel, mod); err != nil {
			log.Print(err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
	// Exit 125 tells gokrazy "don't supervise": do not restart this program.
	os.Exit(125)
}
