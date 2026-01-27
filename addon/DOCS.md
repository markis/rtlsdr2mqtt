# RTL-SDR to MQTT

Read smart utility meters via RTL-SDR and publish readings to MQTT with Home Assistant auto-discovery.

## Requirements

- RTL-SDR USB dongle
- MQTT broker (the Home Assistant Mosquitto add-on works great)

## Supported Protocols

- `scm` - Standard Consumption Message
- `scm+` - SCM Plus (enhanced)
- `idm` - Interval Data Message
- `netidm` - Network IDM
- `r900` - R900 Protocol
- `r900bcd` - R900 BCD Encoded

## Configuration

### General Settings

| Option | Description | Default |
|--------|-------------|---------|
| `sleep_for` | Seconds between reading cycles (0 = continuous) | `0` |
| `verbosity` | Log level (debug, info, warning, error, none) | `info` |
| `health_check_enabled` | Enable health check endpoint | `true` |

### SDR Settings

| Option | Description | Default |
|--------|-------------|---------|
| `usb_device` | USB device ID (BUS:DEV format), empty for auto-detect | empty |
| `freq_correction` | PPM frequency correction | `0` |
| `gain_mode` | Gain mode (auto or manual) | `auto` |
| `gain` | Gain in tenths of dB (e.g., 496 = 49.6 dB) | `0` |
| `agc_enabled` | Enable RTL2832 AGC | `true` |

### MQTT Settings

If you're using the Mosquitto add-on, the MQTT connection details are auto-configured.

| Option | Description | Default |
|--------|-------------|---------|
| `host` | MQTT broker hostname | auto |
| `port` | MQTT broker port | `1883` |
| `user` | MQTT username | auto |
| `password` | MQTT password | auto |
| `base_topic` | Base MQTT topic | `meters` |

### Meter Configuration

Add your meters to the `meters` list:

```yaml
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
    unit_of_measurement: gal
    icon: mdi:water
    device_class: water
    state_class: total_increasing
```

### Meter Options

| Option | Description | Default |
|--------|-------------|---------|
| `id` | Meter ID to match (required) | - |
| `protocol` | Meter protocol (required) | - |
| `name` | Display name | `Smart Meter` |
| `format` | Number format mask (e.g., `######.###`) | none |
| `unit_of_measurement` | Unit for Home Assistant | `kWh` |
| `icon` | MDI icon | `mdi:flash` |
| `device_class` | HA device class (energy, gas, water, none) | `energy` |
| `state_class` | HA state class | `total_increasing` |
| `expire_after` | Seconds until unavailable (0 = disabled) | `0` |
| `force_update` | Always publish updates | `false` |

## Finding Your Meter ID

Set `verbosity` to `debug` and check the add-on logs. You'll see all meter readings being received. Look for your meter's ID in the output and add it to your configuration.

## Troubleshooting

### No meters detected

1. Ensure your RTL-SDR dongle is plugged in
2. Check that the correct protocol is selected
3. Try adjusting the gain settings
4. Verify the dongle is visible in **Supervisor > System > Hardware**

### USB device not found

1. Restart the add-on after plugging in the dongle
2. Try rebooting Home Assistant
3. Check if another add-on is using the RTL-SDR device
