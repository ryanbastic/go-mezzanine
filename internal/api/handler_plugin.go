package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/ryanbastic/go-mezzanine/internal/trigger"
)

// --- Huma Input/Output types ---

type RegisterPluginBody struct {
	Name              string   `json:"name" doc:"Plugin name" required:"true" minLength:"1"`
	Endpoint          string   `json:"endpoint" doc:"JSON-RPC endpoint URL" required:"true" minLength:"1"`
	SubscribedColumns []string `json:"subscribed_columns" doc:"Columns to subscribe to" required:"true" minItems:"1"`
}

type RegisterPluginInput struct {
	Body RegisterPluginBody
}

type PluginResponse struct {
	ID                uuid.UUID `json:"id" doc:"Plugin UUID"`
	Name              string    `json:"name" doc:"Plugin name"`
	Endpoint          string    `json:"endpoint" doc:"JSON-RPC endpoint URL"`
	SubscribedColumns []string  `json:"subscribed_columns" doc:"Subscribed columns"`
	Status            string    `json:"status" doc:"Plugin status" example:"active"`
	CreatedAt         time.Time `json:"created_at" doc:"Creation timestamp"`
}

type RegisterPluginOutput struct {
	Body PluginResponse
}

type ListPluginsInput struct{}

type ListPluginsOutput struct {
	Body []PluginResponse
}

type GetPluginInput struct {
	PluginID string `path:"plugin_id" doc:"Plugin UUID" format:"uuid"`
}

type GetPluginOutput struct {
	Body PluginResponse
}

type DeletePluginInput struct {
	PluginID string `path:"plugin_id" doc:"Plugin UUID" format:"uuid"`
}

// --- Handler ---

type PluginHandler struct {
	registry *trigger.PluginRegistry
	logger   *slog.Logger
}

func NewPluginHandler(registry *trigger.PluginRegistry, logger *slog.Logger) *PluginHandler {
	return &PluginHandler{registry: registry, logger: logger}
}

func registerPluginRoutes(api huma.API, h *PluginHandler) {
	huma.Register(api, huma.Operation{
		OperationID:   "register-plugin",
		Method:        http.MethodPost,
		Path:          "/v1/plugins",
		Summary:       "Register a trigger plugin",
		Tags:          []string{"plugins"},
		DefaultStatus: http.StatusCreated,
	}, h.RegisterPlugin)

	huma.Register(api, huma.Operation{
		OperationID: "list-plugins",
		Method:      http.MethodGet,
		Path:        "/v1/plugins",
		Summary:     "List all plugins",
		Tags:        []string{"plugins"},
	}, h.ListPlugins)

	huma.Register(api, huma.Operation{
		OperationID: "get-plugin",
		Method:      http.MethodGet,
		Path:        "/v1/plugins/{plugin_id}",
		Summary:     "Get a plugin by ID",
		Tags:        []string{"plugins"},
	}, h.GetPlugin)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-plugin",
		Method:        http.MethodDelete,
		Path:          "/v1/plugins/{plugin_id}",
		Summary:       "Delete a plugin",
		Tags:          []string{"plugins"},
		DefaultStatus: http.StatusNoContent,
	}, h.DeletePlugin)
}

func (h *PluginHandler) RegisterPlugin(ctx context.Context, input *RegisterPluginInput) (*RegisterPluginOutput, error) {
	p := &trigger.Plugin{
		Name:              input.Body.Name,
		Endpoint:          input.Body.Endpoint,
		SubscribedColumns: input.Body.SubscribedColumns,
	}
	if err := h.registry.Register(p); err != nil {
		return nil, huma.Error409Conflict(err.Error())
	}

	h.logger.Info("plugin registered", "id", p.ID, "name", p.Name, "endpoint", p.Endpoint)

	return &RegisterPluginOutput{Body: pluginToResponse(p)}, nil
}

func (h *PluginHandler) ListPlugins(ctx context.Context, input *ListPluginsInput) (*ListPluginsOutput, error) {
	plugins := h.registry.List()
	resp := make([]PluginResponse, len(plugins))
	for i, p := range plugins {
		resp[i] = pluginToResponse(p)
	}
	return &ListPluginsOutput{Body: resp}, nil
}

func (h *PluginHandler) GetPlugin(ctx context.Context, input *GetPluginInput) (*GetPluginOutput, error) {
	id, err := uuid.Parse(input.PluginID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid plugin_id")
	}

	p, err := h.registry.Get(id)
	if err != nil {
		return nil, huma.Error404NotFound("plugin not found")
	}

	return &GetPluginOutput{Body: pluginToResponse(p)}, nil
}

func (h *PluginHandler) DeletePlugin(ctx context.Context, input *DeletePluginInput) (*struct{}, error) {
	id, err := uuid.Parse(input.PluginID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid plugin_id")
	}

	if err := h.registry.Delete(id); err != nil {
		return nil, huma.Error404NotFound("plugin not found")
	}

	h.logger.Info("plugin deleted", "id", id)
	return nil, nil
}

func pluginToResponse(p *trigger.Plugin) PluginResponse {
	return PluginResponse{
		ID:                p.ID,
		Name:              p.Name,
		Endpoint:          p.Endpoint,
		SubscribedColumns: p.SubscribedColumns,
		Status:            string(p.Status),
		CreatedAt:         p.CreatedAt,
	}
}
