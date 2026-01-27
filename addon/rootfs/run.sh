#!/usr/bin/with-contenv bashio
# shellcheck shell=bash

# Get MQTT configuration from Home Assistant if not set
if bashio::services.available "mqtt"; then
    if ! bashio::config.has_value 'mqtt.host'; then
        export MQTT_HOST="$(bashio::services mqtt 'host')"
    fi
    if ! bashio::config.has_value 'mqtt.user'; then
        export MQTT_USER="$(bashio::services mqtt 'username')"
    fi
    if ! bashio::config.has_value 'mqtt.password'; then
        export MQTT_PASSWORD="$(bashio::services mqtt 'password')"
    fi
fi

bashio::log.info "Starting RTL-SDR to MQTT..."

exec /usr/bin/rtlsdr2mqtt
