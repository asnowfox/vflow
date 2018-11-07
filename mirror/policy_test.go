package mirror

import (
	"testing"
)

func TestLoadPolicy(t *testing.T) {
	LoadPolicy("../scripts/policy.conf")
}
