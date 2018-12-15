package snmp

import (
	"testing"
	"fmt"
	"strings"
)

func TestReadConfig(t *testing.T) {
	if strings.Contains("dddd{\"i\":8,\"v\":\"0.0.0","{\"i\":8,\"v\":\"0.0.0"){
		fmt.Printf("contains")
	}
	fmt.Print("hello\r\n")
}
