package app

import (
	goccyyaml "github.com/goccy/go-yaml"
	yamlv4 "go.yaml.in/yaml/v4"
)

// Anchor parser dependencies selected for the first runnable milestone until
// analyzer packages own these imports directly.
var (
	_ = goccyyaml.Unmarshal
	_ = yamlv4.Unmarshal
)
