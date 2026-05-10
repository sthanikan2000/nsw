package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OpenNSW/nsw-task-flow/orchestrator"
	tfstore "github.com/OpenNSW/nsw-task-flow/store"
)

// OGARouter exposes the /api/oga/* surface that the React oga-app expects.
//
// In the OpenNSW reference architecture each agency runs its own OGA service
// (FCAU, NPQS, …) that manages its officer-side queue. For this backend's
// demo we collapse everything into the NSW backend itself: the same
// nsw-task-flow TaskRecord rows that drive the trader-app are reshaped here
// for the oga-app, and reviews are completed via TaskManager.CompleteTaskStep
// (which then advances the workflow and wakes the parent).
type OGARouter struct {
	tm       *orchestrator.TaskManager
	registry *orchestrator.TaskTemplateRegistry
}

func NewOGARouter(tm *orchestrator.TaskManager, registry *orchestrator.TaskTemplateRegistry) *OGARouter {
	return &OGARouter{tm: tm, registry: registry}
}

// ── Wire types ────────────────────────────────────────────────────────────────

type ogaFormSchema struct {
	Schema   any `json:"schema"`
	UISchema any `json:"uiSchema"`
}

type ogaFeedbackEntry struct {
	Content   map[string]any `json:"content"`
	Timestamp string         `json:"timestamp"`
	Round     int            `json:"round"`
}

type ogaApplication struct {
	TaskID          string             `json:"taskId"`
	WorkflowID      string             `json:"workflowId"`
	ServiceURL      string             `json:"serviceUrl"`
	Data            map[string]any     `json:"data"`
	OGAActionData   map[string]any     `json:"ogaActionData,omitempty"`
	DataForm        *ogaFormSchema     `json:"dataForm,omitempty"`
	OGAForm         *ogaFormSchema     `json:"ogaForm,omitempty"`
	Status          string             `json:"status"`
	FeedbackHistory []ogaFeedbackEntry `json:"feedbackHistory,omitempty"`
	ReviewerNotes   string             `json:"reviewerNotes,omitempty"`
	ReviewedAt      string             `json:"reviewedAt,omitempty"`
	CreatedAt       string             `json:"createdAt"`
	UpdatedAt       string             `json:"updatedAt"`
}

type ogaWorkflowSummary struct {
	WorkflowID string `json:"workflowId"`
	UpdatedAt  string `json:"updatedAt"`
	Status     string `json:"status"`
	TaskCount  int    `json:"taskCount"`
}

type ogaPaginated[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// ── Endpoints ────────────────────────────────────────────────────────────────

// HandleListWorkflows backs GET /api/oga/workflows. Groups TaskRecords by
// parent_workflow_id and returns one row per workflow with a pending-task count.
func (o *OGARouter) HandleListWorkflows(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page, pageSize := paginationParams(r)

	all := o.tm.GetAllTasks()
	type acc struct {
		latest    time.Time
		taskCount int
		hasOpen   bool
	}
	groups := map[string]*acc{}
	for _, rec := range all {
		if rec.ParentWorkflowID == "" {
			continue
		}
		if q != "" && !strings.Contains(rec.ParentWorkflowID, q) && !strings.Contains(rec.TaskID, q) {
			continue
		}
		g, ok := groups[rec.ParentWorkflowID]
		if !ok {
			g = &acc{}
			groups[rec.ParentWorkflowID] = g
		}
		if rec.CreatedAt.After(g.latest) {
			g.latest = rec.CreatedAt
		}
		g.taskCount++
		if rec.Status == "QUEUED_EXTERNALLY" {
			g.hasOpen = true
		}
	}

	items := make([]ogaWorkflowSummary, 0, len(groups))
	for wid, g := range groups {
		status := "COMPLETED"
		if g.hasOpen {
			status = "PENDING"
		}
		items = append(items, ogaWorkflowSummary{
			WorkflowID: wid,
			UpdatedAt:  g.latest.UTC().Format(time.RFC3339),
			Status:     status,
			TaskCount:  g.taskCount,
		})
	}
	// Stable order: newest first.
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })

	o.writePaginated(w, items, page, pageSize)
}

// HandleListApplications backs GET /api/oga/applications. Returns a paginated
// list of TaskRecords filtered by status (default PENDING) and optional workflowId.
func (o *OGARouter) HandleListApplications(w http.ResponseWriter, r *http.Request) {
	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	workflowFilter := strings.TrimSpace(r.URL.Query().Get("workflowId"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page, pageSize := paginationParams(r)

	all := o.tm.GetAllTasks()
	apps := make([]ogaApplication, 0, len(all))
	for _, rec := range all {
		// Only tasks that have an active OGA-side step are reviewable.
		regEntry, _ := o.registry.Get(rec.ActiveTaskTemplateID)
		if !pluginNeedsOGA(regEntry.PluginName) {
			continue
		}

		appStatus := mapTaskStatusToOGAStatus(rec.Status)
		if statusFilter != "" && appStatus != statusFilter {
			continue
		}
		if workflowFilter != "" && rec.ParentWorkflowID != workflowFilter {
			continue
		}
		if q != "" && !strings.Contains(rec.TaskID, q) && !strings.Contains(rec.ParentWorkflowID, q) {
			continue
		}

		apps = append(apps, o.buildApplication(rec, regEntry))
	}

	sort.Slice(apps, func(i, j int) bool { return apps[i].UpdatedAt > apps[j].UpdatedAt })

	o.writePaginated(w, apps, page, pageSize)
}

// HandleGetApplication backs GET /api/oga/applications/{taskId}.
func (o *OGARouter) HandleGetApplication(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "taskId is required")
		return
	}
	rec, ok := o.tm.GetTask(taskID)
	if !ok {
		writeError(w, http.StatusNotFound, "task "+taskID+" not found")
		return
	}
	regEntry, _ := o.registry.Get(rec.ActiveTaskTemplateID)
	app := o.buildApplication(rec, regEntry)
	writeJSON(w, http.StatusOK, app)
}

// HandleSubmitReview backs POST /api/oga/applications/{taskId}/review.
// Wraps the form values under "reviewerform" and forwards to CompleteTaskStep.
func (o *OGARouter) HandleSubmitReview(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "taskId is required")
		return
	}
	var form map[string]any
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if form == nil {
		form = map[string]any{}
	}

	// Expose every form value both under "reviewerform" (for sub-workflows whose
	// output_mapping reads `reviewerform.foo`) and at the top level (for sub-workflows
	// whose output_mapping reads `foo` directly — e.g. the issue-certificate flow's
	// `certificate_id` / `certificate_url` mapping).
	payload := map[string]any{"reviewerform": form}
	for k, v := range form {
		if _, exists := payload[k]; exists {
			continue
		}
		payload[k] = v
	}
	// doc_review_result is required by the shipping-docs sub-workflow's output_mapping.
	// Set it from review_outcome so mapTaskOutputs can find it in the task result.
	if outcome, ok := form["review_outcome"].(string); ok && outcome != "" {
		payload["doc_review_result"] = outcome
	}
	if err := o.tm.CompleteTaskStep(r.Context(), taskID, payload); err != nil {
		o.writeStepError(w, taskID, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("review submitted for %s", taskID),
	})
}

// HandleSubmitFeedback backs POST /api/oga/applications/{taskId}/feedback.
//
// The OpenNSW review_outcome enum is `approve | reject | needs_more_info`.
// "Request changes" maps to `needs_more_info`, which makes the workflow's
// review_gateway loop back to applicant_submission. Any text the officer
// supplies is carried as `rejection_reason`.
func (o *OGARouter) HandleSubmitFeedback(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "taskId is required")
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	feedbackText, _ := body["feedback"].(string)
	reviewer := map[string]any{"review_outcome": "needs_more_info"}
	if feedbackText != "" {
		reviewer["rejection_reason"] = feedbackText
	}

	if err := o.tm.CompleteTaskStep(r.Context(), taskID, map[string]any{
		"reviewerform":      reviewer,
		"doc_review_result": "needs_more_info",
	}); err != nil {
		o.writeStepError(w, taskID, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": fmt.Sprintf("feedback recorded for %s", taskID),
	})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// pluginNeedsOGA returns true for plugins whose active step requires officer
// action (rather than trader action).
func pluginNeedsOGA(pluginName string) bool {
	switch pluginName {
	case "generic_external_review", "generic_officer_input", "register_task_and_wait", "generic_http_post":
		return true
	default:
		return false
	}
}

// mapTaskStatusToOGAStatus maps nsw-task-flow's status enum to the
// PENDING/APPROVED/REJECTED/FEEDBACK_REQUESTED set the oga-app expects.
func mapTaskStatusToOGAStatus(s string) string {
	switch s {
	case "QUEUED_EXTERNALLY":
		return "PENDING"
	case "COMPLETED":
		return "APPROVED"
	case "FAILED":
		return "REJECTED"
	default:
		return strings.ToUpper(s)
	}
}

// buildApplication constructs an ogaApplication payload from a TaskRecord.
// dataForm is the trader's submitted form schema; ogaForm is the officer's
// review schema; both are pulled from the registry's generic templates.
func (o *OGARouter) buildApplication(rec tfstore.TaskRecord, regEntry orchestrator.TaskTemplateEntry) ogaApplication {
	app := ogaApplication{
		TaskID:     rec.TaskID,
		WorkflowID: rec.ParentWorkflowID,
		ServiceURL: extractServiceURL(regEntry.PluginProperties),
		Data:       rec.Data,
		Status:     mapTaskStatusToOGAStatus(rec.Status),
		CreatedAt:  rec.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  rec.CreatedAt.UTC().Format(time.RFC3339),
	}

	// OGA review form (officer fills this in).
	if rec.ReviewerFormID != "" {
		if schema := lookupFormSchema(o.registry, rec.ReviewerFormID); schema != nil {
			app.OGAForm = &ogaFormSchema{
				Schema:   schema["schema"],
				UISchema: schema["uiSchema"],
			}
		}
	}

	// Trader-submitted form, surfaced read-only.
	if rec.UserFormID != "" {
		if schema := lookupFormSchema(o.registry, rec.UserFormID); schema != nil {
			app.DataForm = &ogaFormSchema{
				Schema:   schema["schema"],
				UISchema: schema["uiSchema"],
			}
		}
	}

	// Surface the trader's submitted data (record.Data["userform"]) under
	// `data` so the read-only form panel renders it.
	if userform, ok := rec.Data["userform"].(map[string]any); ok {
		app.Data = userform
	}

	return app
}

// extractServiceURL looks at well-known fields in plugin_properties to
// surface a "where's the external system" hint.
func extractServiceURL(pluginProps json.RawMessage) string {
	if len(pluginProps) == 0 {
		return ""
	}
	var props map[string]any
	if err := json.Unmarshal(pluginProps, &props); err != nil {
		return ""
	}
	for _, key := range []string{"external_url", "url", "payment_service_url"} {
		if v, ok := props[key].(string); ok {
			return v
		}
	}
	return ""
}

func paginationParams(r *http.Request) (page, pageSize int) {
	page = 1
	pageSize = 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 200 {
			pageSize = v
		}
	}
	return page, pageSize
}

func (o *OGARouter) writePaginated(w http.ResponseWriter, items any, page, pageSize int) {
	// items is a slice — count and slice with reflect-free type switch.
	switch s := items.(type) {
	case []ogaApplication:
		writeJSON(w, http.StatusOK, ogaPaginated[ogaApplication]{
			Items:    sliceWindow(s, page, pageSize),
			Total:    len(s),
			Page:     page,
			PageSize: pageSize,
		})
	case []ogaWorkflowSummary:
		writeJSON(w, http.StatusOK, ogaPaginated[ogaWorkflowSummary]{
			Items:    sliceWindow(s, page, pageSize),
			Total:    len(s),
			Page:     page,
			PageSize: pageSize,
		})
	default:
		writeJSON(w, http.StatusOK, items)
	}
}

func sliceWindow[T any](s []T, page, pageSize int) []T {
	start := (page - 1) * pageSize
	if start >= len(s) {
		return []T{}
	}
	end := start + pageSize
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}

func (o *OGARouter) writeStepError(w http.ResponseWriter, taskID string, err error) {
	switch {
	case strings.Contains(err.Error(), "not found"):
		writeError(w, http.StatusNotFound, err.Error())
	case strings.Contains(err.Error(), "already completed"):
		writeError(w, http.StatusConflict, err.Error())
	default:
		slog.Error("taskv2 oga: complete step failed", "taskId", taskID, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// Discard unused-import warning if no helper references context; HandleSubmitReview
// uses r.Context() indirectly so this is just a guard.
var _ = context.Background
