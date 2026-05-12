package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
)

type envelope struct {
	OK          bool             `json:"ok"`
	Command     string           `json:"command"`
	Data        any              `json:"data,omitempty"`
	Diagnostics []diagnostic     `json:"diagnostics,omitempty"`
	Errors      []responseError  `json:"errors,omitempty"`
	Security    securityBoundary `json:"security"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type diagnostic struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source"`
	File     string `json:"file,omitempty"`
}

type securityBoundary struct {
	ReadOnly              bool   `json:"readOnly"`
	FixtureBacked         bool   `json:"fixtureBacked"`
	NetworkAccess         bool   `json:"networkAccess"`
	DockerInvoked         bool   `json:"dockerInvoked"`
	CrossplaneCLIInvoked  bool   `json:"crossplaneCliInvoked"`
	ClusterAccess         bool   `json:"clusterAccess"`
	WritesWorkspace       bool   `json:"writesWorkspace"`
	TrustMode             string `json:"trustMode"`
	ExternalExecutionMode string `json:"externalExecutionMode"`
}

type compositionSummary struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	File             string         `json:"file"`
	Mode             string         `json:"mode"`
	CompositeTypeRef schemaRef      `json:"compositeTypeRef"`
	Pipeline         []pipelineStep `json:"pipeline"`
}

type pipelineStep struct {
	Step        string `json:"step"`
	FunctionRef string `json:"functionRef"`
	InputKind   string `json:"inputKind"`
	TemplateRef string `json:"templateRef,omitempty"`
}

type schemaRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

type schemaLookup struct {
	Query   schemaRef      `json:"query"`
	Found   bool           `json:"found"`
	Schema  *schemaSummary `json:"schema,omitempty"`
	Matches []schemaRef    `json:"matches,omitempty"`
}

type schemaSummary struct {
	ID             string        `json:"id"`
	File           string        `json:"file"`
	SourceKind     string        `json:"sourceKind"`
	Referenceable  bool          `json:"referenceable"`
	Served         bool          `json:"served"`
	Fields         []schemaField `json:"fields"`
	ProvenanceNote string        `json:"provenanceNote"`
}

type schemaField struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type validationResult struct {
	Workspace string   `json:"workspace"`
	Valid     bool     `json:"valid"`
	Checked   []string `json:"checked"`
	Limits    []string `json:"limits"`
}

type renderResult struct {
	FixtureBacked   bool               `json:"fixtureBacked"`
	Authoritative   bool               `json:"authoritative"`
	Inputs          renderInputs       `json:"inputs"`
	Resources       []renderedResource `json:"resources"`
	FunctionResults []functionResult   `json:"functionResults"`
	Execution       renderExecution    `json:"execution"`
}

type renderInputs struct {
	XR          string `json:"xr"`
	Composition string `json:"composition"`
	Functions   string `json:"functions"`
}

type renderedResource struct {
	APIVersion              string         `json:"apiVersion"`
	Kind                    string         `json:"kind"`
	Name                    string         `json:"name"`
	CompositionResourceName string         `json:"compositionResourceName"`
	Ready                   bool           `json:"ready"`
	Fields                  map[string]any `json:"fields"`
}

type functionResult struct {
	Step     string `json:"step"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type renderExecution struct {
	Mode                  string `json:"mode"`
	DockerInvoked         bool   `json:"dockerInvoked"`
	CrossplaneCLIInvoked  bool   `json:"crossplaneCliInvoked"`
	NetworkAccess         bool   `json:"networkAccess"`
	ClusterAccess         bool   `json:"clusterAccess"`
	SanitizedMetadataOnly bool   `json:"sanitizedMetadataOnly"`
}

var fixtureCompositions = []compositionSummary{
	{
		ID:   "composition.platform.example.org.xbuckets",
		Name: "xbuckets.platform.example.org",
		File: "fixtures/composition-bucket.yaml",
		Mode: "Pipeline",
		CompositeTypeRef: schemaRef{
			APIVersion: "platform.example.org/v1alpha1",
			Kind:       "XBucket",
		},
		Pipeline: []pipelineStep{
			{
				Step:        "render-bucket",
				FunctionRef: "function-go-templating",
				InputKind:   "GoTemplate",
				TemplateRef: "fixtures/templates/bucket.yaml.gotmpl",
			},
			{
				Step:        "auto-ready",
				FunctionRef: "function-auto-ready",
				InputKind:   "Input",
			},
		},
	},
}

var fixtureSchemas = map[schemaRef]schemaSummary{
	{APIVersion: "platform.example.org/v1alpha1", Kind: "XBucket"}: {
		ID:            "xrd.platform.example.org.xbuckets.v1alpha1",
		File:          "fixtures/xrd-bucket.yaml",
		SourceKind:    "CompositeResourceDefinition",
		Referenceable: true,
		Served:        true,
		Fields: []schemaField{
			{
				Path:        "spec.parameters.region",
				Type:        "string",
				Required:    true,
				Description: "Target cloud region for the composed bucket.",
			},
			{
				Path:        "spec.parameters.versioning",
				Type:        "boolean",
				Required:    false,
				Description: "Whether object versioning should be enabled.",
			},
			{
				Path:        "status.bucketName",
				Type:        "string",
				Required:    false,
				Description: "Observed bucket name surfaced by the composition.",
			},
		},
		ProvenanceNote: "Static spike fixture; no schema download, package pull, or cluster discovery was performed.",
	},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(args []string, out io.Writer) int {
	if len(args) == 0 {
		return writeJSON(out, errorEnvelope("", "missing-command", "missing command"))
	}

	command := args[0]
	switch command {
	case "list-compositions":
		return runListCompositions(command, args[1:], out)
	case "find-schema":
		return runFindSchema(command, args[1:], out)
	case "validate-workspace":
		return runValidateWorkspace(command, args[1:], out)
	case "render":
		return runRender(command, args[1:], out)
	default:
		return writeJSON(out, errorEnvelope(command, "unknown-command", fmt.Sprintf("unknown command %q", command)))
	}
}

func runListCompositions(command string, args []string, out io.Writer) int {
	fs := newFlagSet(command)
	if err := fs.Parse(args); err != nil {
		return writeJSON(out, errorEnvelope(command, "invalid-args", err.Error()))
	}
	if fs.NArg() != 0 {
		return writeJSON(out, errorEnvelope(command, "invalid-args", "list-compositions does not accept positional arguments"))
	}

	return writeJSON(out, envelope{
		OK:      true,
		Command: command,
		Data: map[string]any{
			"workspace":    ".",
			"compositions": fixtureCompositions,
		},
		Security: defaultSecurity(),
	})
}

func runFindSchema(command string, args []string, out io.Writer) int {
	fs := newFlagSet(command)
	apiVersion := fs.String("api-version", "", "apiVersion to resolve")
	kind := fs.String("kind", "", "kind to resolve")
	if err := fs.Parse(args); err != nil {
		return writeJSON(out, errorEnvelope(command, "invalid-args", err.Error()))
	}
	if fs.NArg() != 0 {
		return writeJSON(out, errorEnvelope(command, "invalid-args", "find-schema does not accept positional arguments"))
	}
	if *apiVersion == "" || *kind == "" {
		return writeJSON(out, errorEnvelope(command, "invalid-args", "find-schema requires --api-version and --kind"))
	}

	query := schemaRef{APIVersion: *apiVersion, Kind: *kind}
	schema, found := fixtureSchemas[query]
	lookup := schemaLookup{
		Query: query,
		Found: found,
	}
	if found {
		lookup.Schema = &schema
	} else {
		lookup.Matches = knownSchemaRefs()
	}

	return writeJSON(out, envelope{
		OK:      true,
		Command: command,
		Data:    lookup,
		Diagnostics: []diagnostic{
			{
				Severity: "info",
				Message:  "schema lookup used only local spike fixtures",
				Source:   "agent-api-spike",
			},
		},
		Security: defaultSecurity(),
	})
}

func runValidateWorkspace(command string, args []string, out io.Writer) int {
	fs := newFlagSet(command)
	workspace := fs.String("workspace", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return writeJSON(out, errorEnvelope(command, "invalid-args", err.Error()))
	}
	if fs.NArg() != 0 {
		return writeJSON(out, errorEnvelope(command, "invalid-args", "validate-workspace does not accept positional arguments"))
	}

	return writeJSON(out, envelope{
		OK:      true,
		Command: command,
		Data: validationResult{
			Workspace: *workspace,
			Valid:     true,
			Checked: []string{
				"composition pipeline shape",
				"compositeTypeRef to XRD schema fixture",
				"go-templating template reference",
				"render fixture input set",
			},
			Limits: []string{
				"no filesystem traversal beyond fixture metadata",
				"no provider CRD indexing",
				"no Docker, Crossplane CLI, network, or cluster access",
			},
		},
		Security: defaultSecurity(),
	})
}

func runRender(command string, args []string, out io.Writer) int {
	fs := newFlagSet(command)
	xr := fs.String("xr", "fixtures/xr-bucket.yaml", "XR input path")
	composition := fs.String("composition", "fixtures/composition-bucket.yaml", "Composition input path")
	functions := fs.String("functions", "fixtures/functions.yaml", "Function package input path")
	if err := fs.Parse(args); err != nil {
		return writeJSON(out, errorEnvelope(command, "invalid-args", err.Error()))
	}
	if fs.NArg() != 0 {
		return writeJSON(out, errorEnvelope(command, "invalid-args", "render does not accept positional arguments"))
	}

	return writeJSON(out, envelope{
		OK:      true,
		Command: command,
		Data: renderResult{
			FixtureBacked: true,
			Authoritative: false,
			Inputs: renderInputs{
				XR:          *xr,
				Composition: *composition,
				Functions:   *functions,
			},
			Resources: []renderedResource{
				{
					APIVersion:              "s3.aws.upbound.io/v1beta1",
					Kind:                    "Bucket",
					Name:                    "demo-bucket",
					CompositionResourceName: "bucket",
					Ready:                   true,
					Fields: map[string]any{
						"spec.forProvider.region": "us-east-1",
						"spec.forProvider.tags": map[string]string{
							"managed-by": "vibe-xpls-spike",
						},
					},
				},
			},
			FunctionResults: []functionResult{
				{
					Step:     "render-bucket",
					Severity: "Normal",
					Message:  "fixture render emitted one composed bucket",
				},
				{
					Step:     "auto-ready",
					Severity: "Normal",
					Message:  "fixture render marked composed resource ready",
				},
			},
			Execution: renderExecution{
				Mode:                  "fixture",
				DockerInvoked:         false,
				CrossplaneCLIInvoked:  false,
				NetworkAccess:         false,
				ClusterAccess:         false,
				SanitizedMetadataOnly: true,
			},
		},
		Diagnostics: []diagnostic{
			{
				Severity: "info",
				Message:  "render result is fixture-backed and not authoritative Crossplane execution",
				Source:   "agent-api-spike",
			},
		},
		Security: defaultSecurity(),
	})
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func defaultSecurity() securityBoundary {
	return securityBoundary{
		ReadOnly:              true,
		FixtureBacked:         true,
		NetworkAccess:         false,
		DockerInvoked:         false,
		CrossplaneCLIInvoked:  false,
		ClusterAccess:         false,
		WritesWorkspace:       false,
		TrustMode:             "untrusted-workspace-safe",
		ExternalExecutionMode: "disabled",
	}
}

func errorEnvelope(command, code, message string) envelope {
	return envelope{
		OK:      false,
		Command: command,
		Errors: []responseError{
			{
				Code:    code,
				Message: message,
			},
		},
		Security: defaultSecurity(),
	}
}

func writeJSON(out io.Writer, response envelope) int {
	if err := json.NewEncoder(out).Encode(response); err != nil {
		if errors.Is(err, io.ErrClosedPipe) {
			return 1
		}
		return 1
	}
	if response.OK {
		return 0
	}
	return 2
}

func knownSchemaRefs() []schemaRef {
	refs := make([]schemaRef, 0, len(fixtureSchemas))
	for ref := range fixtureSchemas {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].APIVersion == refs[j].APIVersion {
			return refs[i].Kind < refs[j].Kind
		}
		return refs[i].APIVersion < refs[j].APIVersion
	})
	return refs
}
