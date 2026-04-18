package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
)

const dockerFileContents = `
FROM debian:bookworm

RUN apt-get update && apt-get install -y crossbuild-essential-armhf bc libssl-dev bison flex

COPY gokr-build-kernel /usr/bin/gokr-build-kernel
{{- range $idx, $path := .Patches }}
COPY {{ $path }} /usr/src/{{ $path }}
{{- end }}

# Stage firmware for embedding via CONFIG_EXTRA_FIRMWARE
RUN mkdir -p /tmp/firmware
COPY regulatory.db /tmp/firmware/regulatory.db
COPY regulatory.db.p7s /tmp/firmware/regulatory.db.p7s

RUN echo 'builduser:x:{{ .Uid }}:{{ .Gid }}:nobody:/:/bin/sh' >> /etc/passwd && \
    chown -R {{ .Uid }}:{{ .Gid }} /usr/src /tmp/firmware

USER builduser
WORKDIR /usr/src
ENTRYPOINT /usr/bin/gokr-build-kernel
`

var dockerFileTmpl = template.Must(template.New("dockerfile").
	Funcs(map[string]interface{}{
		"basename": func(path string) string {
			return filepath.Base(path)
		},
	}).
	Parse(dockerFileContents))

var patchFiles = []string{
	// Add patch filenames here as needed
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

var gopath = mustGetGopath()

func mustGetGopath() string {
	gopathb, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		log.Panic(err)
	}
	return strings.TrimSpace(string(gopathb))
}

func find(filename string) (string, error) {
	if _, err := os.Stat(filename); err == nil {
		return filename, nil
	}

	path := filepath.Join(gopath, "src", "github.com", "consolving", "gokrazy-kernel-a20", filename)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("could not find file %q (looked in . and %s)", filename, path)
}

func getContainerExecutable() (string, error) {
	choices := []string{"podman", "docker"}
	for _, exe := range choices {
		p, err := exec.LookPath(exe)
		if err != nil {
			continue
		}
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			return "", err
		}
		return resolved, nil
	}
	return "", fmt.Errorf("none of %v found in $PATH", choices)
}

func main() {
	var overwriteContainerExecutable = flag.String("overwrite_container_executable",
		"",
		"E.g. docker or podman to overwrite the automatically detected container executable")
	flag.Parse()
	executable, err := getContainerExecutable()
	if err != nil {
		log.Fatal(err)
	}
	if *overwriteContainerExecutable != "" {
		executable = *overwriteContainerExecutable
	}
	execName := filepath.Base(executable)

	tmp, err := os.MkdirTemp("/tmp", "gokr-rebuild-kernel")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	cmd := exec.Command("go", "install", "github.com/consolving/gokrazy-kernel-a20/cmd/gokr-build-kernel")
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "GOBIN="+tmp)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("%v: %v", cmd.Args, err)
	}

	buildPath := filepath.Join(tmp, "gokr-build-kernel")

	var patchPaths []string
	for _, filename := range patchFiles {
		path, err := find(filename)
		if err != nil {
			log.Fatal(err)
		}
		patchPaths = append(patchPaths, path)
	}

	kernelPath, err := find("vmlinuz")
	if err != nil {
		log.Fatal(err)
	}
	dtbPath, err := find("sun7i-a20-lamobo-r1.dtb")
	if err != nil {
		log.Fatal(err)
	}

	for _, path := range patchPaths {
		if err := copyFile(filepath.Join(tmp, filepath.Base(path)), path); err != nil {
			log.Fatal(err)
		}
	}

	// Copy regulatory.db for embedding into the kernel via CONFIG_EXTRA_FIRMWARE
	regdbPath, err := find(filepath.Join("lib", "firmware", "regulatory.db"))
	if err != nil {
		log.Printf("warning: regulatory.db not found, skipping: %v", err)
	} else {
		if err := copyFile(filepath.Join(tmp, "regulatory.db"), regdbPath); err != nil {
			log.Fatal(err)
		}
	}
	regdbP7sPath, err := find(filepath.Join("lib", "firmware", "regulatory.db.p7s"))
	if err != nil {
		log.Printf("warning: regulatory.db.p7s not found, skipping: %v", err)
	} else {
		if err := copyFile(filepath.Join(tmp, "regulatory.db.p7s"), regdbP7sPath); err != nil {
			log.Fatal(err)
		}
	}

	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dockerFile, err := os.Create(filepath.Join(tmp, "Dockerfile"))
	if err != nil {
		log.Fatal(err)
	}

	if err := dockerFileTmpl.Execute(dockerFile, struct {
		Uid       string
		Gid       string
		BuildPath string
		Patches   []string
	}{
		Uid:       u.Uid,
		Gid:       u.Gid,
		BuildPath: buildPath,
		Patches:   patchFiles,
	}); err != nil {
		log.Fatal(err)
	}

	if err := dockerFile.Close(); err != nil {
		log.Fatal(err)
	}

	log.Printf("building %s container for kernel compilation", execName)

	dockerBuild := exec.Command(execName,
		"build",
		"--rm=true",
		"--tag=gokr-rebuild-kernel-a20",
		".")
	dockerBuild.Dir = tmp
	dockerBuild.Stdout = os.Stdout
	dockerBuild.Stderr = os.Stderr
	if err := dockerBuild.Run(); err != nil {
		log.Fatalf("%s build: %v (cmd: %v)", execName, err, dockerBuild.Args)
	}

	log.Printf("compiling kernel")

	var dockerRun *exec.Cmd
	if execName == "podman" {
		dockerRun = exec.Command(executable,
			"run",
			"--userns=keep-id",
			"--rm",
			"--volume", tmp+":/tmp/buildresult:Z",
			"gokr-rebuild-kernel-a20")
	} else {
		dockerRun = exec.Command(executable,
			"run",
			"--rm",
			"--volume", tmp+":/tmp/buildresult:Z",
			"gokr-rebuild-kernel-a20")
	}
	dockerRun.Dir = tmp
	dockerRun.Stdout = os.Stdout
	dockerRun.Stderr = os.Stderr
	if err := dockerRun.Run(); err != nil {
		log.Fatalf("%s run: %v (cmd: %v)", execName, err, dockerRun.Args)
	}

	if err := copyFile(kernelPath, filepath.Join(tmp, "vmlinuz")); err != nil {
		log.Fatal(err)
	}

	if err := copyFile(dtbPath, filepath.Join(tmp, "sun7i-a20-lamobo-r1.dtb")); err != nil {
		log.Fatal(err)
	}

	// Copy kernel modules (lib/modules/) to the repo
	repoDir := filepath.Dir(kernelPath)
	libSrc := filepath.Join(tmp, "lib")
	if _, err := os.Stat(libSrc); err == nil {
		libDest := filepath.Join(repoDir, "lib")

		// Preserve firmware files before wiping lib/
		fwBackup, err := os.MkdirTemp("", "fw-backup")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(fwBackup)
		fwDir := filepath.Join(libDest, "firmware")
		hasFirmware := false
		if _, err := os.Stat(fwDir); err == nil {
			hasFirmware = true
			cpFw := exec.Command("cp", "-a", fwDir, filepath.Join(fwBackup, "firmware"))
			if err := cpFw.Run(); err != nil {
				log.Fatalf("backing up firmware: %v", err)
			}
		}

		os.RemoveAll(libDest)
		cpCmd := exec.Command("cp", "-a", libSrc, libDest)
		cpCmd.Stdout = os.Stdout
		cpCmd.Stderr = os.Stderr
		if err := cpCmd.Run(); err != nil {
			log.Fatalf("copying modules: %v", err)
		}
		log.Printf("kernel modules installed to %s", libDest)

		// Restore firmware files
		if hasFirmware {
			restCmd := exec.Command("cp", "-a", filepath.Join(fwBackup, "firmware"), fwDir)
			if err := restCmd.Run(); err != nil {
				log.Fatalf("restoring firmware: %v", err)
			}
			log.Printf("restored firmware files to %s", fwDir)
		}
	}
}
