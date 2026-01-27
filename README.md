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
| `icon` | MDI icon | `mdi:flash` |
| `device_class` | HA device class (`energy`, `gas`, `water`) | `energy` |
| `state_class` | HA state class | `total_increasing` |
| `expire_after` | Seconds until unavailable | 0 (disabled) |
| `force_update` | Always publish updates | `false` |

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
