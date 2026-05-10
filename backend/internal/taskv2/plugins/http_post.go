package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// HTTPPostPlugin replaces nsw-task-flow's generic_http_post. Fire-and-forget:
// posts the task data to the configured URL and transitions to DISPATCHED.
type HTTPPostPlugin struct {
	client *dispatchClient
}

func NewHTTPPostPlugin(backendBaseURL string, devMode bool) *HTTPPostPlugin {
	return &HTTPPostPlugin{client: newDispatchClient(backendBaseURL, devMode)}
}

func (p *HTTPPostPlugin) Name() string { return "generic_http_post" }

type httpPostConfig struct {
	URL      string `json:"url"`
	TaskCode string `json:"task_code,omitempty"`
}

func (p *HTTPPostPlugin) Execute(ctx pluginContext, configRaw json.RawMessage) error {
	var cfg httpPostConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return fmt.Errorf("http_post: invalid config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("http_post: url is required")
	}

	ctx.Record.Status = "DISPATCHED"

	body := buildSubmissionBody(ctx.Record, cfg.TaskCode, p.client.callbackTasksURL())

	slog.Info("taskv2 http_post: dispatching fire-and-forget",
		"taskId", ctx.Record.TaskID, "url", cfg.URL, "taskCode", cfg.TaskCode)

	return p.client.post(ctx.Context, cfg.URL, body)
}
