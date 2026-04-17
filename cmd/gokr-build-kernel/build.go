package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	_ "embed"
)

//go:embed config.txt
var configContents []byte

// see https://www.kernel.org/releases.json
var latest = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.12.23.tar.xz"

func downloadKernel() error {
	out, err := os.Create(filepath.Base(latest))
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(latest)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		return fmt.Errorf("unexpected HTTP status code for %s: got %d, want %d", latest, got, want)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}
	return out.Close()
}

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
	defconfig := exec.Command("make", "ARCH=arm", "sunxi_defconfig")
	defconfig.Stdout = os.Stdout
	defconfig.Stderr = os.Stderr
	if err := defconfig.Run(); err != nil {
		return fmt.Errorf("make defconfig: %v", err)
	}

	f, err := os.OpenFile(".config", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(configContents); err != nil {
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

	make := exec.Command("make", "zImage", "dtbs", "modules", "-j"+strconv.Itoa(runtime.NumCPU()))
	make.Env = append(os.Environ(),
		"ARCH=arm",
		"CROSS_COMPILE=arm-linux-gnueabihf-",
		"KBUILD_BUILD_USER=gokrazy",
		"KBUILD_BUILD_HOST=docker",
		"KBUILD_BUILD_TIMESTAMP=Wed Mar  1 20:57:29 UTC 2017",
	)
	make.Stdout = os.Stdout
	make.Stderr = os.Stderr
	if err := make.Run(); err != nil {
		return fmt.Errorf("make: %v", err)
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
	log.Printf("downloading kernel source: %s", latest)
	if err := downloadKernel(); err != nil {
		log.Fatal(err)
	}

	log.Printf("unpacking kernel source")
	untar := exec.Command("tar", "xf", filepath.Base(latest))
	untar.Stdout = os.Stdout
	untar.Stderr = os.Stderr
	if err := untar.Run(); err != nil {
		log.Fatalf("untar: %v", err)
	}

	srcdir := strings.TrimSuffix(filepath.Base(latest), ".tar.xz")

	log.Printf("applying patches")
	if err := applyPatches(srcdir); err != nil {
		log.Fatal(err)
	}

	if err := os.Chdir(srcdir); err != nil {
		log.Fatal(err)
	}

	log.Printf("compiling kernel")
	if err := compile(); err != nil {
		log.Fatal(err)
	}

	if err := copyFile("/tmp/buildresult/vmlinuz", "arch/arm/boot/zImage"); err != nil {
		log.Fatal(err)
	}

	if err := copyFile("/tmp/buildresult/sun7i-a20-lamobo-r1.dtb", "arch/arm/boot/dts/allwinner/sun7i-a20-lamobo-r1.dtb"); err != nil {
		log.Fatal(err)
	}

	// Install kernel modules
	log.Printf("installing kernel modules")
	modInstall := exec.Command("make",
		"ARCH=arm",
		"CROSS_COMPILE=arm-linux-gnueabihf-",
		"INSTALL_MOD_PATH=/tmp/modstaging",
		"modules_install",
	)
	modInstall.Stdout = os.Stdout
	modInstall.Stderr = os.Stderr
	if err := modInstall.Run(); err != nil {
		log.Fatalf("make modules_install: %v", err)
	}

	// Copy lib/modules/ tree to build result
	cpMods := exec.Command("cp", "-a", "/tmp/modstaging/lib", "/tmp/buildresult/lib")
	cpMods.Stdout = os.Stdout
	cpMods.Stderr = os.Stderr
	if err := cpMods.Run(); err != nil {
		log.Fatalf("copying modules: %v", err)
	}

	// Remove build/source symlinks (point to build host paths)
	modDirs, _ := filepath.Glob("/tmp/buildresult/lib/modules/*/build")
	for _, d := range modDirs {
		os.Remove(d)
	}
	modDirs, _ = filepath.Glob("/tmp/buildresult/lib/modules/*/source")
	for _, d := range modDirs {
		os.Remove(d)
	}
}
