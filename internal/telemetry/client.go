package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/docker/docker-language-server/internal/configuration"
	"github.com/docker/docker-language-server/internal/pkg/cli/metadata"
	"github.com/dromara/carbon/v2"
)

const apiKey = "eIxc3dSmud2vuJRKiq9hJ6wORVWfoLxp1nqb4qXz"

type TelemetryClient interface {
	Enqueue(event string, properties map[string]any)
	Publish(ctx context.Context) (int, error)
	UpdateTelemetrySetting(value string)
}

type TelemetryClientImpl struct {
	mutex     sync.Mutex
	telemetry configuration.TelemetrySetting
	records   []Record
}

const telemetryUrl = "https://api.docker.com/events/v1/track"

func NewClient() TelemetryClient {
	return &TelemetryClientImpl{telemetry: configuration.TelemetrySettingAll}
}

func (c *TelemetryClientImpl) UpdateTelemetrySetting(value string) {
	switch value {
	case "all":
		c.telemetry = configuration.TelemetrySettingAll
	case "error":
		c.telemetry = configuration.TelemetrySettingError
	case "off":
		c.telemetry = configuration.TelemetrySettingOff
	default:
		c.telemetry = configuration.TelemetrySettingAll
	}
}

func (c *TelemetryClientImpl) allow(err bool) bool {
	if c.telemetry == configuration.TelemetrySettingAll {
		return true
	}

	return c.telemetry == configuration.TelemetrySettingError && err
}

func (c *TelemetryClientImpl) Enqueue(event string, properties map[string]any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	value, ok := properties["type"].(string)
	if c.allow(ok && event == EventServerHeartbeat && value == ServerHeartbeatTypePanic) {
		c.records = append(c.records, Record{
			Event:      event,
			Source:     "editor_integration",
			Properties: properties,
			Timestamp:  carbon.Now().TimestampMilli(),
		})
	}
}

func (c *TelemetryClientImpl) trimRecords() []Record {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	records := c.records
	if len(c.records) > 500 {
		records = c.records[0:500]
		c.records = c.records[500:]
	} else {
		c.records = nil
	}
	return records
}

func (c *TelemetryClientImpl) Publish(ctx context.Context) (int, error) {
	if os.Getenv("DOCKER_LANGUAGE_SERVER_TELEMETRY") == "false" {
		c.records = nil
		return 0, nil
	}

	if len(c.records) == 0 {
		return 0, nil
	}

	records := c.trimRecords()

	payload := &TelemetryPayload{Records: records}
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal telemetry payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telemetryUrl, bytes.NewBuffer(b))
	if err != nil {
		return 0, fmt.Errorf("failed to create telemetry request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("docker-language-server/v%v", metadata.Version))
	req.Header.Set("x-api-key", apiKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send telemetry request: %w", err)
	}

	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 400 {
		return 0, fmt.Errorf("telemetry http request failed (%v status code)", res.StatusCode)
	}
	return len(records), nil
}
