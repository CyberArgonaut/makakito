// Package experiment provides YAML loading and validation for experiment files.
package experiment

import (
	"fmt"
	"os"

	"github.com/CyberArgonaut/makakito/pkg/schema"
	"gopkg.in/yaml.v3"
)

// Load reads an experiment file from disk and parses it.
func Load(path string) (*schema.Experiment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading experiment file %q: %w", path, err)
	}
	return Parse(data)
}

// Parse unmarshals experiment YAML bytes into an Experiment.
func Parse(data []byte) (*schema.Experiment, error) {
	var exp schema.Experiment
	if err := yaml.Unmarshal(data, &exp); err != nil {
		return nil, fmt.Errorf("parsing experiment YAML: %w", err)
	}
	return &exp, nil
}
