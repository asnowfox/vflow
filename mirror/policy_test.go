package mirror

import (
	"testing"
	"fmt"
)

func TestLoadPolicy(t *testing.T) {
	LoadPolicy("../scripts/policy.conf")
	for _,policy := range policyConfigs {
		fmt.Printf("policy: %s\n",policy.PolicyId)
		for _,p := range policy.Rules {
			fmt.Printf("  source %s, inport %d,outport %d, dst %s\n",
				p.Source,p.Port,p.Direction,p.DistAddress)
		}
	}
}
