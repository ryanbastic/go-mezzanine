package trigger

import (
	"context"
	"log/slog"

	"github.com/ryanbastic/go-mezzanine/internal/cell"
)

// Notifier dispatches cell-write notifications to subscribed plugins via JSON-RPC.
type Notifier struct {
	registry  *PluginRegistry
	rpcClient *RPCClient
	logger    *slog.Logger
}

// NewNotifier creates a Notifier.
func NewNotifier(registry *PluginRegistry, rpcClient *RPCClient, logger *slog.Logger) *Notifier {
	return &Notifier{
		registry:  registry,
		rpcClient: rpcClient,
		logger:    logger,
	}
}

// NotifyCell fires a goroutine per subscribed plugin to deliver a cell.written
// JSON-RPC notification. Errors are logged, not propagated — writes are never
// blocked by slow plugins.
func (n *Notifier) NotifyCell(shardID int, c *cell.Cell) {
	plugins := n.registry.ForColumn(c.ColumnName)
	if len(plugins) == 0 {
		return
	}

	params := CellWrittenParams{
		AddedID:    c.AddedID,
		RowKey:     c.RowKey.String(),
		ColumnName: c.ColumnName,
		RefKey:     c.RefKey,
		Body:       c.Body,
		CreatedAt:  c.CreatedAt,
		ShardID:    shardID,
	}

	for _, p := range plugins {
		go func(endpoint, pluginName string) {
			resp, err := n.rpcClient.Call(context.Background(), endpoint, "cell.written", params)
			if err != nil {
				n.logger.Error("trigger rpc failed", "plugin", pluginName, "endpoint", endpoint, "error", err)
				return
			}
			if resp.Error != nil {
				n.logger.Error("trigger rpc returned error", "plugin", pluginName, "endpoint", endpoint, "error", resp.Error)
			}
		}(p.Endpoint, p.Name)
	}
}
