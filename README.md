# rtlsdr2mqtt

RTL-SDR to MQTT bridge for smart utility meters with Home Assistant integration.

Reads smart meter data using an RTL-SDR dongle and publishes readings to MQTT with automatic Home Assistant discovery.

## Supported Protocols

- `scm` - Standard Consumption Message
- `scm+` - SCM Plus (enhanced)
- `idm` - Interval Data Message
- `netidm` - Network IDM
- `r900` - R900 Protocol
- `r900bcd` - R900 BCD Encoded

## Quick Start

### Docker

```bash
docker run -d \
  --name rtlsdr2mqtt \
  --device /dev/bus/usb \
  -v /path/to/config.yaml:/etc/rtlsdr2mqtt.yaml \
  ghcr.io/markis/rtlsdr2mqtt:latest
```

### Docker Compose

```yaml
services:
  rtlsdr2mqtt:
    image: ghcr.io/markis/rtlsdr2mqtt:latest
    devices:
      - /dev/bus/usb:/dev/bus/usb
    volumes:
      - ./config.yaml:/etc/rtlsdr2mqtt.yaml
    restart: unless-stopped
```

## Configuration

Configuration can be provided via YAML or JSON. The application searches for config files in this order:

1. `/data/options.json` (Home Assistant add-on)
2. `/data/options.yaml`
3. `/data/options.yml`
4. `/etc/rtlsdr2mqtt.yaml`
5. Path specified by `-config` flag

### Example Configuration

```yaml
general:
  sleep_for: 0              # Seconds between reading cycles (0 = continuous)
  verbosity: info           # debug, info, warning, error, none
  health_check_enabled: true

sdr:
  usb_device: ""            # USB device ID (BUS:DEV format), empty for auto-detect
  freq_correction: 0        # PPM frequency correction
  gain_mode: auto           # auto or manual
  gain: 0                   # Gain in tenths of dB (e.g., 496 = 49.6 dB)
  agc_enabled: true         # RTL2832 AGC
  device_retry_seconds: 5   # Re-enumeration poll cap (seconds, >= 1) for USB recovery

mqtt:
  host: localhost
  port: 1883
  user: ""
  password: ""
  base_topic: meters
  tls:
    enabled: false
    insecure: false
    ca: ""
    cert: ""
    keyfile: ""
  homeassistant:
    enabled: true
    discovery_prefix: homeassistant
    status_topic: homeassistant/status

meters:
  - id: "12345678"
    protocol: scm+
    name: Electric Meter
    unit_of_measurement: kWh
    icon: mdi:flash
    device_class: energy
    state_class: total_increasing

  - id: "87654321"
    protocol: r900
    name: Water Meter
    format: "######.###"
    unit_of_measurement: "gal"
    icon: mdi:water
    device_class: water
    state_class: total_increasing
```

### Meter Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `id` | Meter ID to match | (required) |
| `protocol` | Meter protocol | `scm+` |
| `name` | Display name | `Smart Meter` |
| `format` | Number format mask (e.g., `######.###`) | none |
| `unit_of_measurement` | Unit for Home Assistant | `kWh` |
| `icon` | [Icon for Home Assistant](https://pictogrammers.com/library/mdi/) | `mdi:flash` |
| `device_class` | [Home Assistant device class](https://www.home-assistant.io/integrations/sensor/#device-class) (`energy`, `gas`, `water`) | `energy` |
| `state_class` | [Home Assistant state class](https://developers.home-assistant.io/docs/core/entity/sensor/#available-state-classes) | `total_increasing` |
| `expire_after` | Seconds until unavailable | 0 (disabled) |
| `force_update` | Always publish updates | `false` |

## Reliability Behavior

The application self-recovers from upstream outages instead of waiting for the container health check to restart it:

- **MQTT publishes are bounded.** Publishes, subscribes, and unsubscribes time out after 10 seconds. During a broker outage, readings still log and the health check heartbeat (touched on every decoded reading) stays fresh; failed publishes are logged as errors and retried on the next reading.
- **Sample-flow watchdog.** If the RTL-SDR stops delivering samples for 30 seconds (wedged USB pipe, dead dongle), the decoder stops the stale stream and enters USB recovery, with an error-level log naming the cause.
- **Unexpected stream closure.** If the device stops on its own, the decoder enters USB recovery.
- **USB re-enumeration recovery.** When the dongle drops off the USB bus (for example an RTL2838 that re-enumerates at a different `/dev/bus/usb` node), the app detects the loss, closes the stale libusb handle, and polls for the device to reappear by fresh USB enumeration (VID:PID based, never the old node path). Polls back off 1s → 2s → 5s → 10s, capped at `sdr.device_retry_seconds` (default 5s, minimum 1s). On success it reopens the device, reapplies the full startup configuration (center frequency, sample rate, gain/AGC, PPM), resumes decoding in-process, and touches the health check file once. Expected logs:
  - `level=WARN msg="RTL-SDR lost; entering re-enumeration recovery" error=<cause>`
  - `level=INFO msg="RTL-SDR recovery attempt failed; retrying" attempt=<n> retry_in=<delay> error=<error>` (one line per failed attempt)
  - `level=INFO msg="RTL-SDR re-acquired" elapsed=<duration> device=<identity> attempt=<n>`
- **Health check file.** `HEALTHCHECK_FILE` is touched on each decoded reading and once per successful device (re-)acquisition. The stock `scripts/healthcheck.sh` fails when the file is older than 5 minutes. This is the last-resort backstop: it only trips if self-recovery has already failed.

Limits: recovery selects the same logical device startup would (configured index, else first available RTL-SDR); with several matching dongles the first enumerated one wins. A physically present but unresponsive dongle may keep its USB claim held, in which case reopening fails until the device actually re-enumerates; the liveness probe then restarts the container. Software recovery hides dropout symptoms, not the underlying hardware cause — cable, port, hub, and power issues are still worth fixing.

## Development

```bash
# Setup development environment
make setup-dev

# Build
make build

# Run tests
make test

# Lint
make lint

# See all available targets
make
```

## Acknowledgments

Special thanks to these projects that made this possible:

- [rtlamr2mqtt](https://github.com/allangood/rtlamr2mqtt) - Original Python implementation
- [rtlamr](https://github.com/bemasher/rtlamr) - RTL-SDR AMR meter receiver
- [rtl-sdr](https://osmocom.org/projects/rtl-sdr/wiki/Rtl-sdr) - RTL-SDR library and tools

## License

AGPL-3.0
