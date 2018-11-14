//: ----------------------------------------------------------------------------
//: Copyright (C) 2017 Verizon.  All Rights Reserved.
//: All Rights Reserved
//:
//: file:    marshal.go
//: details: encoding of each decoded netflow v9 data sets
//: author:  Mehrdad Arshad Rad
//: date:    04/27/2017
//:
//: Licensed under the Apache License, Version 2.0 (the "License");
//: you may not use this file except in compliance with the License.
//: You may obtain a copy of the License at
//:
//:     http://www.apache.org/licenses/LICENSE-2.0
//:
//: Unless required by applicable law or agreed to in writing, software
//: distributed under the License is distributed on an "AS IS" BASIS,
//: WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//: See the License for the specific language governing permissions and
//: limitations under the License.
//: ----------------------------------------------------------------------------

package netflow9

import (
	"bytes"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
)

var errUknownMarshalDataType = errors.New("unknown data type to marshal")

// JSONMarshal encodes netflow v9 message
func (m *Message) JSONMarshal(b *bytes.Buffer, datas []DataFlowRecord) ([]byte, error) {

	b.WriteString("{")

	// encode agent id
	m.encodeAgent(b)

	// encode header
	m.encodeHeader(b)

	// encode data sets
	if err := m.encodeDataSet(b,datas); err != nil {
		return nil, err
	}

	b.WriteString("}")

	return b.Bytes(), nil
}

func (m *Message) encodeDataSet(b *bytes.Buffer,datas []DataFlowRecord) error {
	var (
		length   int
		dsLength int
		err      error
	)

	b.WriteString("\"DataSets\":")
	dsLength = len(datas)

	b.WriteByte('[')

	for i,data := range datas {
		length = len(data.DataSets)

		b.WriteByte('[')
		for j := range data.DataSets {
			b.WriteString("{\"I\":")
			b.WriteString(strconv.FormatInt(int64(data.DataSets[j].ID), 10))
			b.WriteString(",\"V\":")
			err = m.writeValue(b, j,data.DataSets)

			if j < length-1 {
				b.WriteString("},")
			} else {
				b.WriteByte('}')
			}
		}

		if i < dsLength-1 {
			b.WriteString("],")
		} else {
			b.WriteByte(']')
		}
	}

	b.WriteByte(']')

	return err
}



func (m *Message) encodeHeader(b *bytes.Buffer) {
	b.WriteString("\"Header\":{\"Version\":")
	b.WriteString(strconv.FormatInt(int64(m.Header.Version), 10))
	b.WriteString(",\"Count\":")
	b.WriteString(strconv.FormatInt(int64(m.Header.Count), 10))
	b.WriteString(",\"SysUpTime\":")
	b.WriteString(strconv.FormatInt(int64(m.Header.SysUpTime), 10))
	b.WriteString(",\"UNIXSecs\":")
	b.WriteString(strconv.FormatInt(int64(m.Header.UNIXSecs), 10))
	b.WriteString(",\"SeqNum\":")
	b.WriteString(strconv.FormatInt(int64(m.Header.SeqNum), 10))
	b.WriteString(",\"SrcID\":")
	b.WriteString(strconv.FormatInt(int64(m.Header.SrcID), 10))
	b.WriteString("},")
}

func (m *Message) encodeAgent(b *bytes.Buffer) {
	b.WriteString("\"AgentID\":\"")
	b.WriteString(m.AgentID)
	b.WriteString("\",")
}

func (m *Message) writeValue(b *bytes.Buffer, i int, datas []DecodedField) error {
	switch datas[i].Value.(type) {
	case uint:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(uint)), 10))
	case uint8:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(uint8)), 10))
	case uint16:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(uint16)), 10))
	case uint32:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(uint32)), 10))
	case uint64:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(uint64)), 10))
	case int:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(int)), 10))
	case int8:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(int8)), 10))
	case int16:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(int16)), 10))
	case int32:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(int32)), 10))
	case int64:
		b.WriteString(strconv.FormatInt(int64(datas[i].Value.(int64)), 10))
	case float32:
		b.WriteString(strconv.FormatFloat(float64(datas[i].Value.(float32)), 'E', -1, 32))
	case float64:
		b.WriteString(strconv.FormatFloat(datas[i].Value.(float64), 'E', -1, 64))
	case string:
		b.WriteByte('"')
		b.WriteString(datas[i].Value.(string))
		b.WriteByte('"')
	case net.IP:
		b.WriteByte('"')
		b.WriteString(datas[i].Value.(net.IP).String())
		b.WriteByte('"')
	case net.HardwareAddr:
		b.WriteByte('"')
		b.WriteString(datas[i].Value.(net.HardwareAddr).String())
		b.WriteByte('"')
	case []uint8:
		b.WriteByte('"')
		b.WriteString("0x" + hex.EncodeToString(datas[i].Value.([]uint8)))
		b.WriteByte('"')
	default:
		return errUknownMarshalDataType
	}

	return nil
}
