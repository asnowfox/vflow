package utils

import (
	"bytes"
	"encoding/gob"
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"
)

func HostAddrCheck(addr string) (bool, error) {
	items := strings.Split(addr, ":")
	if items == nil || len(items) != 2 {
		return false, errors.New("must split with ':'")
	}

	a := net.ParseIP(items[0])
	if a == nil {
		return false, errors.New("can not parse ip " + items[0])
	}

	match, err := regexp.MatchString("^[0-9]*$", items[1])
	if err != nil {
		return false, errors.New("port " + items[1] + " is not a number.")
	}

	i, err := strconv.Atoi(items[1])
	if err != nil {
		return false, errors.New("port " + items[1] + " is not a number.")
	}
	if i < 0 || i > 65535 {
		return false, errors.New("port " + items[1] + " is illegal, too big or to small")
	}

	if match == false {
		return false, errors.New("port " + items[1] + " is illegal.")
	}

	return true, nil
}

func RuleCheck(rule Rule) (bool, error) {
	a := net.ParseIP(rule.Source)
	if a == nil {
		return false, errors.New("can not parse ip " + rule.Source)
	}
	if rule.Port < int32(-1) || rule.Port > int32(65535) {
		return false, errors.New("port is illegal, too big or to small")
	}

	if rule.Direction != 0 && rule.Direction != 1 && rule.Direction != -1 {
		return false, errors.New("direction is error must be 0 1 or -l")
	}

	return true, nil
}

func MQNameCheck(dstQueueName string) (bool, error) {
	result, _ := regexp.MatchString("^\\w+$", dstQueueName)
	if !result {
		return false, errors.New("queue name must be character number and _")
	}
	return true, nil
}

func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}
