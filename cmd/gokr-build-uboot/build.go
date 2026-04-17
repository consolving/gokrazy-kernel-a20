package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

// U-Boot v2025.04 (latest stable as of 2025)
const ubootRev = "v2025.04"
const ubootTS = 1600000000

const uBootRepo = "https://github.com/u-boot/u-boot"

func applyPatches(srcdir string) error {
	patches, err := filepath.Glob("*.patch")
	if err != nil {
		return err
	}
	for _, patch := range patches {
		log.Printf("applying patch %q", patch)
		f, err := os.Open(patch)
		if err != nil {
			return err
		}
		defer f.Close()
		cmd := exec.Command("patch", "-p1")
		cmd.Dir = srcdir
		cmd.Stdin = f
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
		f.Close()
	}
	return nil
}

func compile() error {
	defconfig := exec.Command("make", "ARCH=arm", "Lamobo_R1_defconfig")
	defconfig.Stdout = os.Stdout
	defconfig.Stderr = os.Stderr
	if err := defconfig.Run(); err != nil {
		return fmt.Errorf("make defconfig: %v", err)
	}

	// Append CONFIG_CMD_SETEXPR for boot.cmd support
	f, err := os.OpenFile(".config", os.O_RDWR|os.O_APPEND, 0755)
	if err != nil {
		return err
	}
	if _, err := f.Write([]byte("CONFIG_CMD_SETEXPR=y\nCONFIG_CMD_SETEXPR_FMT=y\n")); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	olddefconfig := exec.Command("make", "ARCH=arm", "olddefconfig")
	olddefconfig.Stdout = os.Stdout
	olddefconfig.Stderr = os.Stderr
	if err := olddefconfig.Run(); err != nil {
		return fmt.Errorf("make olddefconfig: %v", err)
	}

	make := exec.Command("make", "-j"+strconv.Itoa(runtime.NumCPU()))
	make.Env = append(os.Environ(),
		"ARCH=arm",
		"CROSS_COMPILE=arm-linux-gnueabihf-",
		"SOURCE_DATE_EPOCH="+strconv.Itoa(ubootTS),
	)
	make.Stdout = os.Stdout
	make.Stderr = os.Stderr
	if err := make.Run(); err != nil {
		return fmt.Errorf("make: %v", err)
	}

	return nil
}

func generateBootScr(bootCmdPath string) error {
	mkimage := exec.Command("./tools/mkimage", "-A", "arm", "-T", "script", "-C", "none", "-d", bootCmdPath, "boot.scr")
	mkimage.Env = append(os.Environ(),
		"ARCH=arm",
		"CROSS_COMPILE=arm-linux-gnueabihf-",
		"SOURCE_DATE_EPOCH=1600000000",
	)
	mkimage.Stdout = os.Stdout
	mkimage.Stderr = os.Stderr
	if err := mkimage.Run(); err != nil {
		return fmt.Errorf("mkimage: %v", err)
	}
	return nil
}

func copyFile(dest, src string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	st, err := in.Stat()
	if err != nil {
		return err
	}
	if err := out.Chmod(st.Mode()); err != nil {
		return err
	}
	return out.Close()
}

func main() {
	ubootDir, err := os.MkdirTemp("", "u-boot")
	if err != nil {
		log.Fatal(err)
	}

	var bootCmdPath string
	if p, err := filepath.Abs("boot.cmd"); err != nil {
		log.Fatal(err)
	} else {
		bootCmdPath = p
	}

	if err := os.Chdir(ubootDir); err != nil {
		log.Fatal(err)
	}

	// Clone U-Boot at the specified tag
	for _, cmd := range [][]string{
		{"git", "init"},
		{"git", "remote", "add", "origin", uBootRepo},
		{"git", "fetch", "--depth=1", "origin", ubootRev},
		{"git", "checkout", "FETCH_HEAD"},
	} {
		log.Printf("Running %s", cmd)
		cmdObj := exec.Command(cmd[0], cmd[1:]...)
		cmdObj.Stdout = os.Stdout
		cmdObj.Stderr = os.Stderr
		cmdObj.Dir = ubootDir
		if err := cmdObj.Run(); err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("applying patches")
	if err := applyPatches(ubootDir); err != nil {
		log.Fatal(err)
	}

	log.Printf("compiling uboot")
	if err := compile(); err != nil {
		log.Fatal(err)
	}

	log.Printf("generating boot.scr")
	if err := generateBootScr(bootCmdPath); err != nil {
		log.Fatal(err)
	}

	// A20 U-Boot produces u-boot-sunxi-with-spl.bin (SPL + U-Boot combined)
	for _, copyCfg := range []struct {
		dest, src string
	}{
		{"boot.scr", "boot.scr"},
		{"u-boot-sunxi-with-spl.bin", "u-boot-sunxi-with-spl.bin"},
	} {
		if err := copyFile(filepath.Join("/tmp/buildresult", copyCfg.dest), copyCfg.src); err != nil {
			log.Fatal(err)
		}
	}
}
