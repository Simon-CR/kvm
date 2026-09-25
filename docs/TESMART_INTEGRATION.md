# TESmart KVM Switch Integration for Luckfox PicoKVM

Native TCP integration connecting Luckfox PicoKVM directly to TESmart HDMI/DisplayPort KVM switches (e.g., 4-port, 8-port, and 16-port models) over local network without requiring external bridge scripts or home automation hubs.

---

## Features

- **Quick Header Channel Selector:** Switch between inputs directly from the viewer navigation bar next to Power and Terminal buttons.
- **Custom Port Naming:** Assign friendly names to every input (e.g., `Port 1: TrueNAS`, `Port 2: Proxmox Node 1`, `Port 3: Gaming PC`).
- **Real-Time Port Sync:** Automatically polls the active port via TCP so manual physical button presses on the TESmart hardware update the web UI instantly.
- **Keyboard Hotkeys:** Rapid switching with `Ctrl + Alt + [1-8]`.
- **Preflight Connection Testing:** Dedicated "Test Connection" button in Settings with round-trip latency readout.
- **TESmart Network Configuration:** View current Switch IP and Gateway over TCP port 5000 and push updated IP settings directly from the Web UI.
- **Clean Fallback & Zero Disruption:** Non-blocking network timeouts ensure video streaming and mouse/keyboard HID remain completely unaffected if the switch is powered off or unreachable.

---

## One-Line Installation (For Stock OEM PicoKVM)

SSH into your PicoKVM box as `root` and run:

```sh
curl -sSL https://raw.githubusercontent.com/Simon-CR/kvm/feature/tesmart-kvm-switch/install.sh | sh
```

The script will:
1. Verify device compatibility (`Rockchip RV1106`).
2. Back up original OEM binaries (`/userdata/picokvm/bin/kvm_app.oem.bak` and `/usr/bin/kvm_app.oem.bak`).
3. Deploy the enhanced binary to both locations to satisfy the cold-boot persistence invariant.
4. Restart `kvm_app` cleanly.

### Uninstallation

To restore stock OEM behavior at any time:

```sh
curl -sSL https://raw.githubusercontent.com/Simon-CR/kvm/feature/tesmart-kvm-switch/install.sh | sh -s -- --uninstall
```

---

## Configuration

Navigate to **Settings** -> **KVM Switch** in the PicoKVM web interface:

1. **Switch IP & Port:** Enter the TESmart switch IP (factory default is typically `192.168.1.10`) and TCP port (`5000`).
2. **Total Ports:** Select your switch capacity (2, 4, 8, or 16).
3. **Port Labels:** Set custom names for each port.
4. **Hotkeys:** Toggle `Ctrl + Alt + [1-8]` keyboard switching.
5. **Connection Test:** Click "Test Connection" to confirm live reachability.
6. **Network Configuration (Optional):** Click "Refresh" to query the switch's current IP and Gateway, or update the IP address over the network.

---

## Technical Details

### TCP Communication Protocol
- **Switch Port:** `0xAA 0xBB 0x03 0x01 <port> 0xEE`
- **Query Active Port:** `0xAA 0xBB 0x03 0x10 0x00 0xEE`
  - Response: `0xAA 0xBB 0x03 0x11 <port> <checksum>` (buffered 6-byte stream read)
- **ASCII Network Configuration:**
  - Query IP: `IP?\r\n` -> `IP:192.168.001.010;`
  - Query Gateway: `GW?\r\n` -> `GW:192.168.001.001;`
  - Set IP / Gateway: `IP:<new_ip>;\r\n`, `GW:<new_gw>;\r\n`

### REST API Endpoints
- `GET /api/kvm-switch/config` - Retrieve current configuration
- `POST /api/kvm-switch/config` - Save configuration
- `GET /api/kvm-switch/status` - Read active port
- `POST /api/kvm-switch/select` - Switch to port `{"port": N}`
- `POST /api/kvm-switch/test` - Test reachability and latency
- `GET /api/kvm-switch/network` - Query switch IP and Gateway
- `POST /api/kvm-switch/network` - Push new network parameters to switch

---

## Building From Source

```sh
# Build Web UI
cd ui && npm ci && npm run build:device && cd ..

# Cross-compile for Rockchip RV1106 (ARMv7)
GOOS=linux GOARCH=arm GOARM=7 go build -tags netgo -trimpath \
  -ldflags="-s -w -X kvm.builtAppVersion=0.1.4-tesmart" \
  -o bin/kvm_app cmd/main.go
```
