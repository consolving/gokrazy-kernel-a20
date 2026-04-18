# gokrazy-kernel-a20

Kernel, U-Boot, and device tree for running [gokrazy](https://gokrazy.org/) on the Banana Pi BPI-R1 (Lamobo R1) with Allwinner A20 (sun7i) SoC.

## Usage

Add to your gokrazy instance config:

```json
{
    "DeviceType": "bpi_r1",
    "KernelPackage": "github.com/consolving/gokrazy-kernel-a20",
    "FirmwarePackage": "",
    "EEPROMPackage": "",
    "SerialConsole": "ttyS0,115200"
}
```

Build an SD card image:

```bash
GOARCH=arm gok -i bpi-r1 overwrite --full /dev/sdX
```

## Loading Kernel Modules

gokrazy has no `modprobe`. This package includes `cmd/loadmodules`, which loads
modules at boot via `finit_module(2)`. Modules are passed as arguments (paths
relative to `/lib/modules/<release>/`), loaded in order.

### Example: WiFi (RTL8192CU)

```json
{
    "Packages": [
        "github.com/consolving/gokrazy-kernel-a20/cmd/loadmodules"
    ],
    "PackageConfig": {
        "github.com/consolving/gokrazy-kernel-a20/cmd/loadmodules": {
            "CommandLineFlags": [
                "kernel/drivers/net/wireless/realtek/rtl8xxxu/rtl8xxxu.ko"
            ]
        }
    }
}
```

### Example: Multiple modules with dependencies

List dependencies first:

```json
{
    "PackageConfig": {
        "github.com/consolving/gokrazy-kernel-a20/cmd/loadmodules": {
            "CommandLineFlags": [
                "kernel/lib/crypto/libarc4.ko",
                "kernel/net/wireless/cfg80211.ko",
                "kernel/net/mac80211/mac80211.ko",
                "kernel/drivers/net/wireless/realtek/rtl8xxxu/rtl8xxxu.ko"
            ]
        }
    }
}
```

## Rebuilding

```bash
go run ./cmd/gokr-rebuild-kernel   # requires Docker/Podman
go run ./cmd/gokr-rebuild-uboot
```

Kernel config addendum: [`cmd/gokr-build-kernel/config.txt`](cmd/gokr-build-kernel/config.txt)

## Documentation

- [BPI-R1 Hardware Specification](docs/bpi-r1-hardware.md)
