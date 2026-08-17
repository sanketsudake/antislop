package example

import "encoding/json"

// Rejected by nountypedunmarshal; the variable is also reported by
// noanycontainers.
func ParseSettings(b []byte) error {
	var settings map[string]any
	return json.Unmarshal(b, &settings)
}

// Accepted: the struct names the fields the program reads.
type Settings struct {
	Retries int `json:"retries"`
}

func ParseTypedSettings(b []byte) (Settings, error) {
	var settings Settings
	err := json.Unmarshal(b, &settings)
	return settings, err
}
