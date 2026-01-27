package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	ErrSupervisorBadStatus = errors.New("supervisor API returned bad status")
	ErrSupervisorNoToken   = errors.New("supervisor token not found in environment")
)

// SupervisorMQTTResponse represents the response from the Supervisor API.
type SupervisorMQTTResponse struct {
	Data SupervisorMQTTData `json:"data"`
}

// SupervisorMQTTData represents the MQTT data from the Supervisor API.
type SupervisorMQTTData struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSL      bool   `json:"ssl"`
}

// GetMQTTFromSupervisor fetches MQTT config from HA Supervisor API.
func GetMQTTFromSupervisor() (*MQTTConfig, error) {
	token := os.Getenv("SUPERVISOR_TOKEN")
	if token == "" {
		return nil, ErrSupervisorNoToken
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://supervisor/services/mqtt", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("supervisor API returned status %d and failed to read response body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("%w %d: %s", ErrSupervisorBadStatus, resp.StatusCode, string(body))
	}

	var mqttResp SupervisorMQTTResponse
	if err := json.NewDecoder(resp.Body).Decode(&mqttResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	mqttConfig := &MQTTConfig{
		Host:     mqttResp.Data.Host,
		Port:     mqttResp.Data.Port,
		User:     mqttResp.Data.Username,
		Password: mqttResp.Data.Password,
		TLS: TLSConfig{
			Enabled: mqttResp.Data.SSL,
		},
	}

	return mqttConfig, nil
}
