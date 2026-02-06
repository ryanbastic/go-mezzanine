package trigger

// Registry holds all trigger registrations grouped by column name.
type Registry struct {
	handlers map[string][]HandlerFunc // column_name -> handlers
}

// NewRegistry creates an empty trigger Registry.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string][]HandlerFunc)}
}

// Register adds a handler for a given column name.
func (r *Registry) Register(columnName string, handler HandlerFunc) {
	r.handlers[columnName] = append(r.handlers[columnName], handler)
}

// Columns returns all column names that have registered handlers.
func (r *Registry) Columns() []string {
	cols := make([]string, 0, len(r.handlers))
	for col := range r.handlers {
		cols = append(cols, col)
	}
	return cols
}

// HandlersFor returns the handlers registered for a column name.
func (r *Registry) HandlersFor(columnName string) []HandlerFunc {
	return r.handlers[columnName]
}
