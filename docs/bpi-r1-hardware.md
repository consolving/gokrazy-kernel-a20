# Banana Pi BPI-R1 (Lamobo R1) Hardware Specification

## Overview

The Banana Pi BPI-R1 (also known as Lamobo R1) is an open-source smart router board based on the Allwinner A20 SoC. It is designed for smart home networking use, featuring 5 Gigabit Ethernet ports, onboard WiFi, SATA, and a form factor suited for router/NAS/gateway applications.

- **Manufacturer:** SINOVOIP CO., LIMITED
- **Model:** Lamobo R1 (BPI-R1)
- **Certifications:** CE, FCC, RoHS
- **Product Size:** 148 mm x 100 mm
- **Weight:** 83 g

## SoC and Processing

| Component | Specification |
|-----------|--------------|
| SoC | Allwinner A20 (sun7i) |
| CPU | ARM Cortex-A7 Dual-Core @ 1 GHz |
| GPU | ARM Mali400MP2, OpenGL ES 2.0/1.1 |
| Architecture | ARMv7l (32-bit) |

## Memory and Storage

| Component | Specification |
|-----------|--------------|
| RAM | 1 GB DDR3 (shared with GPU) |
| Onboard Storage | Micro SD card slot (max 64 GB) |
| SATA | 2.5" SATA disk support (up to 2 TB) |

## Networking

| Component | Specification |
|-----------|--------------|
| Ethernet Switch | Broadcom BCM53125 |
| Ethernet Ports | 5x 10/100/1000 Mbps RJ45 (1 WAN + 4 LAN via VLAN) |
| SoC Interface | Single RGMII to BCM53125 |
| WiFi | Realtek RTL8192CU, 802.11 b/g/n, 2T2R MIMO, 300 Mbps |
| WiFi Antennas | 2x detachable external antennas (U.FL connectors) |

### Ethernet Security Note

The BCM53125 interconnects all 5 ports and the A20 SoC at Layer 2 by default. WAN/LAN separation requires VLAN configuration via MDIO. Without proper VLAN setup (during boot, bricked state, or misconfiguration), all ports are bridged together, which is a security concern for router use. The A20 has only a single RGMII interface.

## Video

| Component | Specification |
|-----------|--------------|
| HDMI | Standard HDMI 1.4 (Type A, full-size), 1080p output |
| LVDS/RGB | 40-pin FPC connector (CON2) for LCD panel + I2C touch |
| CSI Camera | 40-pin FPC connector (CON1) for camera module |

## Audio

| Component | Specification |
|-----------|--------------|
| Output | 3.5 mm headphone jack, HDMI audio |
| Input | Onboard microphone |

## USB

| Component | Specification |
|-----------|--------------|
| USB 2.0 Host | 1x USB-A port |
| Micro-USB OTG | 1x (also usable for power input) |

## Power

| Component | Specification |
|-----------|--------------|
| Primary Power | 5V / 2A via Micro-USB (DC in only) |
| Battery | 3.7V lithium battery connector (JST) |
| Power Management | AXP209 PMU |

## Buttons and LEDs

| Component | Specification |
|-----------|--------------|
| Power Button | Next to battery connector |
| Reset Button | Next to power button (top side of PCB) |
| Power LED | Red |
| User LED | Green |
| RJ45 LEDs | Activity/link indicators on Ethernet ports |

## GPIO and Expansion

### CON3 - 26-pin GPIO Header (Raspberry Pi compatible layout)

| Pin | Function | GPIO |
|-----|----------|------|
| 1 | VCC-3V3 | |
| 2 | VCC-5V | |
| 3 | TWI2-SDA | PB21 |
| 4 | VCC-5V | |
| 5 | TWI2-SCK | PB20 |
| 6 | GND | |
| 7 | PWM1 | PI3 |
| 8 | UART3_TX | PH0 |
| 9 | GND | |
| 10 | UART3_RX | PH1 |
| 11 | UART2_RX | PI19 |
| 12 | PH2 | PH2 |
| 13 | UART2_TX | PI18 |
| 14 | GND | |
| 15 | UART2_CTS | PI17 |
| 16 | CAN_TX | PH20 |
| 17 | VCC-3V3 | |
| 18 | CAN_RX | PH21 |
| 19 | SPI0_MOSI | PI12 |
| 20 | GND | |
| 21 | SPI0_MISO | PI13 |
| 22 | UART2_RTS | PI16 |
| 23 | SPI0_CLK | PI11 |
| 24 | SPI0_CS0 | PI10 |
| 25 | GND | |
| 26 | SPI0_CS1 | PI14 |

### J13 - Console UART (UART0)

| Pin | Function | GPIO |
|-----|----------|------|
| 1 | UART0-RX | PB23 |
| 2 | UART0-TX | PB22 |

### J12 - Auxiliary UART + GPIO

| Pin | Function | GPIO |
|-----|----------|------|
| 1 | VCC-5V | |
| 2 | VCC-3V3 | |
| 3 | PH5 | PH5 |
| 4 | UART7_RX | PI21 |
| 5 | PH3 | PH3 |
| 6 | UART7_TX | PI20 |
| 7 | GND | |
| 8 | GND | |

### Bus Interfaces on CON3

- **I2C:** TWI2 (PB20/PB21)
- **SPI:** SPI0 with 2 chip selects (PI10-PI14)
- **CAN Bus:** CAN_TX/CAN_RX (PH20/PH21)
- **UART:** UART2 (with CTS/RTS), UART3
- **PWM:** PWM1 (PI3)

## Other Interfaces

| Interface | Description |
|-----------|------------|
| IR Receiver | Onboard infrared receiver (sunxi-ir) |
| SATA | Onboard SATA connector for 2.5" drives |
| CSI Camera | 40-pin FPC (CON1), 8-bit parallel CSI |
| LVDS/RGB Display | 40-pin FPC (CON2), 24-bit RGB LCD |

## Boot and Software

| Item | Detail |
|------|--------|
| Boot Media | Micro SD card (TF card) |
| Bootloader | U-Boot |
| Device Tree | sun7i-a20-lamobo-r1.dtb |
| Linux Kernel | Mainline sunxi support (linux-sunxi) |
| Supported OS | Linux (Armbian, Bananian, Ubuntu, Debian, Raspbian), OpenWrt, Android 4.4, OpenBSD, NetBSD |
| Board Family | sun7i / sunxi |

### Armbian Reference Configuration

From the Armbian system snapshot in this repository:

```
BOARD=lamobo-r1
BOARD_NAME="Lamobo R1"
BOARDFAMILY=sun7i
LINUXFAMILY=sunxi
ARCH=arm
BRANCH=current
```

### Key Kernel Modules

| Module | Purpose |
|--------|---------|
| 8189fs / rtl8192cu | Realtek WiFi driver |
| b53 | BCM53125 Ethernet switch driver |
| sunxi-ir | Infrared receiver |
| can4linux | CAN bus (optional, community) |

## Key ICs

| IC | Function |
|----|----------|
| Allwinner A20 | Main SoC (dual-core Cortex-A7) |
| Broadcom BCM53125 | 5-port Gigabit Ethernet switch |
| Realtek RTL8192CU | 802.11 b/g/n WiFi (USB-connected) |
| AXP209 | Power management unit |

## References

- linux-sunxi wiki: http://linux-sunxi.org/Lamobo_R1
- Allwinner A20 documentation: http://dl.linux-sunxi.org/A20/
