package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// APICallPlugin implements the "generic_api_call" FIRE_AND_FORGET task type.
// It POSTs the submission envelope to a configured URL, logs the response, and
// transitions the task to DISPATCHED regardless of whether the call succeeds.
type APICallPlugin struct {
	client *dispatchClient
}

func NewAPICallPlugin(backendBaseURL string, devMode bool) *APICallPlugin {
	return &APICallPlugin{client: newDispatchClient(backendBaseURL, devMode)}
}

func (p *APICallPlugin) Name() string { return "generic_api_call" }

type apiCallConfig struct {
	URL string `json:"url"`
}

func (p *APICallPlugin) Execute(ctx pluginContext, configRaw json.RawMessage) error {
	var cfg apiCallConfig
	if err := json.Unmarshal(configRaw, &cfg); err != nil {
		return fmt.Errorf("api_call: invalid config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("api_call: url is required")
	}

	ctx.Record.Status = "DISPATCHED"

	body := buildSubmissionBody(ctx.Record, "", p.client.callbackTasksURL())
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("api_call: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx.Context, http.MethodPost, cfg.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		// TODO: retry failed submissions
		slog.Warn("taskv2 api_call: failed to build request", "taskId", ctx.Record.TaskID, "url", cfg.URL, "error", err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.httpClient.Do(req)
	if err != nil {
		// TODO: retry failed submissions
		slog.Warn("taskv2 api_call: dispatch failed", "taskId", ctx.Record.TaskID, "url", cfg.URL, "error", err)
		return nil
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("taskv2 api_call: failed to close response body", "url", cfg.URL, "error", closeErr)
		}
	}()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	slog.Info("taskv2 api_call: dispatched",
		"taskId", ctx.Record.TaskID, "url", cfg.URL,
		"status", resp.StatusCode, "response", string(respBody))

	return nil
}
