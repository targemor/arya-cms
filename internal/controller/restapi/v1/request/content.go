package request

// ContentEntry is the Strapi-style write envelope. Data is decoded with json.Number so integers
// keep full precision.
type ContentEntry struct {
	Data map[string]any `json:"data" validate:"required"`
} // @name v1.ContentEntryRequest
