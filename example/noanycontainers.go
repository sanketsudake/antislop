package example

import "encoding/json"

// Rejected by noanycontainers.
type Document map[string]any

// Accepted: decode the document at its I/O boundary into a struct.
type DocumentConfig struct {
	Name    string `json:"name"`
	Retries int    `json:"retries"`
}

func DecodeDocumentConfig(data []byte) (DocumentConfig, error) {
	var cfg DocumentConfig
	err := json.Unmarshal(data, &cfg)
	return cfg, err
}
