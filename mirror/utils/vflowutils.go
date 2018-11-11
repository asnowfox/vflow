package utils

import (
	"strings"
	"net"
	"regexp"
	"strconv"
	"errors"
	"../../mirror"
)

func HostAddrCheck(addr string) (bool,error) {
	items := strings.Split(addr, ":")
	if items == nil || len(items) != 2 {
		return false,errors.New("must split with ':'")
	}

	a := net.ParseIP(items[0])
	if a == nil {
		return false,errors.New("can not parse ip "+items[0])
	}

	match, err := regexp.MatchString("^[0-9]*$", items[1])
	if err != nil {
		return false,errors.New("port "+items[1]+" is not a number.")
	}

	i, err := strconv.Atoi(items[1])
	if err != nil {
		return false,errors.New("port "+items[1]+" is not a number.")
	}
	if i < 0 || i > 65535 {
		return false,errors.New("port "+items[1]+" is illegal, too big or to small")
	}

	if match == false {
		return false,errors.New("port "+items[1]+" is illegal.")
	}

	return true,nil
}

func RuleCheck(rule mirror.Rule) (bool,error) {
	a := net.ParseIP(rule.Source)
	if a == nil {
		return false,errors.New("can not parse ip "+rule.Source)
	}
	if rule.InPort < int32(-1) || rule.InPort > int32(65535){
		return false,errors.New("inport is illegal, too big or to small")
	}
	if rule.OutPort < int32(-1) || rule.OutPort > int32(65535){
		return false,errors.New("outport is illegal, too big or to small")
	}
	return true,nil
}
