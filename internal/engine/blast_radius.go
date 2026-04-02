// Package engine provides the core runtime types, interfaces, registries, and
// experiment orchestration logic for makakito.
package engine

import (
	"math/rand"

	"github.com/CyberArgonaut/makakito/pkg/schema"
)

// EnforceBlastRadius caps the resources slice according to the blast radius controls.
// The most restrictive limit wins (max_targets vs percentage).
func EnforceBlastRadius(resources []Resource, controls schema.BlastRadius) []Resource {
	if len(resources) == 0 {
		return resources
	}

	limit := len(resources)

	if controls.MaxTargets > 0 && controls.MaxTargets < limit {
		limit = controls.MaxTargets
	}

	if controls.Percentage > 0 {
		pctLimit := int(float64(len(resources)) * float64(controls.Percentage) / 100.0)
		pctLimit = max(pctLimit, 1)
		if pctLimit < limit {
			limit = pctLimit
		}
	}

	if limit >= len(resources) {
		return resources
	}

	// Random subset selection.
	shuffled := make([]Resource, len(resources))
	copy(shuffled, resources)
	rand.Shuffle(len(shuffled), func(i, j int) { //nolint:gosec // non-cryptographic shuffle is intentional for blast radius sampling
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled[:limit]
}
