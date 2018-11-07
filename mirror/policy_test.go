package mirror

import (
	"testing"
	"fmt"
)

func TestLoadPolicy(t *testing.T) {
	LoadPolicy("../scripts/policy.conf")
	for _,policy := range policyConfigs {
		fmt.Printf("policy:",policy.PolicyId)
		for _,p := range policy.Policies {
			fmt.Printf("  source %s, inport %d,outport %d, dst %s",p.Source,p.InPort,p.OutPort,p.DistAddress)
		}

	}
}
