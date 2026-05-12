package analyzer

import "time"

type Limits struct {
	MaxDocumentBytes     int64
	MaxDiagnosticsPerDoc int
	MaxYAMLFiles         int
	MaxYAMLBytes         int64
	DocumentSoftDeadline time.Duration
}

func DefaultLimits() Limits {
	return Limits{
		MaxDocumentBytes:     2 * 1024 * 1024,
		MaxDiagnosticsPerDoc: 100,
		MaxYAMLFiles:         10000,
		MaxYAMLBytes:         100 * 1024 * 1024,
		DocumentSoftDeadline: 500 * time.Millisecond,
	}
}
