# glassUSB

Create a bootable Windows USB from the terminal 🫗

glassUSB is a simple<!-- cross-platform--> tool for Linux systems to create a bootable Windows USB installer from a Windows ISO / DVD image. It provides both an easy to use CLI and interactive GUI wizard.

## Compatibility

- (Alpha note) *glassUSB has only been tested with Windows 10/11 images in UEFI mode. Legacy BIOS/CSM mode and Windows Vista+ support is not fully tested, since I don't have hardware available.*
- glassUSB only supports 64-bit Windows Vista, 7, 8.x, 10 and 11.\
32-bit Windows and Windows on ARM should work, but are currently untested. If it works for you, let me know!\
Windows XP and earlier are unsupported.
- glassUSB uses an MBR partitioning scheme by default, which supports both UEFI and Legacy BIOS boot modes. GPT partitioning is also supported for UEFI-only boot.

## Usage

### Linux

#### Dependencies

glassUSB requires the following tools to be installed:

- `zenity` — optional, needed for GUI wizard dialogs
- Any of:
  - `ntfs-3g` — NTFS support (recommended)
  - OR `exfatprogs` — exFAT support (note: exFAT is incompatible with Secure Boot)
  - OR `dosfstools` — FAT32 support (note: FAT32 is incompatible with Windows 8+)

These should be preinstalled on most desktop Linux distributions such as Ubuntu and Fedora. If you don't have them, install them via your package manager.

#### Download

Download the latest binary for your system from the [releases page](https://github.com/retrixe/glassusb/releases) using the following commands:

```bash
cd ~/Downloads/ # or wherever you want to download the binary
wget -O glassusb "https://github.com/retrixe/glassUSB/releases/latest/download/glassusb-linux-$(uname -m)"
chmod +x glassusb
```

#### GUI wizard

Run the interactive wizard to flash a Windows ISO to a USB device:

```bash
sudo -E ./glassusb wizard
```

See `glassusb wizard --help` for advanced options, such as using GPT, selecting a custom filesystem, etc.

#### CLI

Flash a Windows ISO to a USB device directly using the CLI:

```bash
# Replace `/path/to/windows.iso` with the path to your Windows ISO file, and
# `/dev/sdX` with your USB device (e.g., `/dev/sdb`). Make sure to double-check
# the device path to avoid data loss.
sudo ./glassusb flash /path/to/windows.iso /dev/sdX
```

See `glassusb flash --help` for advanced options, such as using GPT, selecting a custom filesystem, etc.

<!-- **GUI wizard** — needs your desktop session (D-Bus, display). `sudo -E` preserves those environment variables:

```bash
sudo \
    DBUS_SESSION_BUS_ADDRESS="$DBUS_SESSION_BUS_ADDR
  ESS" \
    XDG_RUNTIME_DIR="$XDG_RUNTIME_DIR" \
    WAYLAND_DISPLAY="$WAYLAND_DISPLAY" \
    DISPLAY="$DISPLAY" \
    XAUTHORITY="${XAUTHORITY:-}" \
    ./glassusb wizard
```

If dialogs fail under `sudo`, pass the session bus explicitly (replace `1000` with your UID from `id -u`):

```bash
sudo DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$(id -u)/bus" ./glassusb wizard
``` -->

### macOS

Currently unsupported, being tracked in [issue #3](https://github.com/retrixe/glassUSB/issues/3)

### Windows

Currently unsupported, but planned in the future. The Linux version should work through WSL2, though. Contributions welcome!

## Additional Notes

- glassUSB cannot create bootable USB drives for Linux distributions. It is only designed to flash Windows ISOs to USB drives.
- glassUSB is not intended for use with CD/DVD drives. If this is a use-case you're interested in, feel free to open an issue.
