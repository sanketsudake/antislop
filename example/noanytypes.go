package example

// Rejected by noanytypes.
type Metadata = any

// Accepted: declare what the values actually carry.
type ReportMetadata struct {
	Owner string
	Tags  []string
}
