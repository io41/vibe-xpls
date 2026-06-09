package schemagen

import "encoding/json"

const coverageFormatVersion = 1

type coverageBucket string

const (
	bucketCoveredUpstream           coverageBucket = "covered-upstream"
	bucketCoveredWithCompatOverride coverageBucket = "covered-with-compat-override"
	bucketCompatAddedField          coverageBucket = "compat-added-field"
	bucketCompatOnlySchema          coverageBucket = "compat-only-schema"
	bucketMissing                   coverageBucket = "missing"
	bucketExcluded                  coverageBucket = "excluded"
	bucketUnsupportedShape          coverageBucket = "unsupported-shape"
)

type metadataCoverageStatus string

const (
	metadataCovered            metadataCoverageStatus = "covered"
	metadataMissing            metadataCoverageStatus = "missing"
	metadataNotPresentUpstream metadataCoverageStatus = "not-present-upstream"
	metadataNotRequired        metadataCoverageStatus = "not-required"
)

type gapCategory string

const (
	gapExcludedResource        gapCategory = "excluded-resource"
	gapUnsupportedOpenAPIShape gapCategory = "unsupported-openapi-shape"
	gapMissingField            gapCategory = "missing-field"
	gapMissingDescription      gapCategory = "missing-description"
	gapMissingType             gapCategory = "missing-type"
	gapMissingRequired         gapCategory = "missing-required"
	gapMissingEnum             gapCategory = "missing-enum"
	gapMissingDefault          gapCategory = "missing-default"
	gapMissingDeprecation      gapCategory = "missing-deprecation"
	gapCompatAddedField        gapCategory = "compat-added-field"
	gapCompatOnlySchema        gapCategory = "compat-only-schema"
)

type coverageProblem struct {
	Message string
}

type observedGap struct {
	Release    string
	APIVersion string
	Kind       string
	Path       string
	Category   gapCategory
	Reason     string
}

type coverageTarget struct {
	Release           string
	APIVersion        string
	Kind              string
	SourcePath        string
	SourceSHA256      string
	Path              string
	Description       string
	Type              string
	Required          bool
	Default           *json.RawMessage
	Enum              []string
	Deprecated        bool
	UnsupportedReason string
}

type actualCoverageField struct {
	Path           string
	Description    string
	Type           string
	Required       bool
	Default        *json.RawMessage
	Enum           []string
	Deprecated     string
	CompatOverride bool
	CompatAdded    bool
}

type coverageFieldState struct {
	Release    string
	APIVersion string
	Kind       string
	Path       string
	Bucket     coverageBucket
	Metadata   coverageMetadataState
	Gap        *observedGap
}

type coverageMetadataState struct {
	Description metadataCoverageStatus `json:"description,omitempty"`
	Type        metadataCoverageStatus `json:"type,omitempty"`
	Required    metadataCoverageStatus `json:"required,omitempty"`
	Enum        metadataCoverageStatus `json:"enum,omitempty"`
	Default     metadataCoverageStatus `json:"default,omitempty"`
	Deprecated  metadataCoverageStatus `json:"deprecated,omitempty"`
}

type coverageGVKState struct {
	Release      string
	APIVersion   string
	Kind         string
	SourcePath   string
	SourceSHA256 string
	Fields       []coverageFieldState
	Buckets      map[coverageBucket]int
}

type coverageState struct {
	GVKs []coverageGVKState
	Gaps []observedGap
}
