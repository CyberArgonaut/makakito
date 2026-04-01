package engine

import (
	"fmt"
	"math/rand"
	"time"
)

// NewExperimentID generates a unique experiment ID.
func NewExperimentID() string {
	ts := time.Now().UTC().Format("20060102-150405")
	r := rand.Int63n(999999) //nolint:gosec
	return fmt.Sprintf("exp-%s-%06d", ts, r)
}
