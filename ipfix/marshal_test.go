//: ----------------------------------------------------------------------------
//: Copyright (C) 2017 Verizon.  All Rights Reserved.
//: All Rights Reserved
//:
//: file:    marshal_test.go
//: details: provides support for automated testing of marshal methods
//: author:  Mehrdad Arshad Rad
//: date:    02/01/2017
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

package ipfix

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"
)

type TestMessage struct {
	AgentID  string
	Header   MessageHeader
	DataSets [][]TestDecodedField
}

type TestDecodedField struct {
	I uint16
	V interface{}
}

var mockDecodedMsg = Message{
	AgentID: "10.10.10.10",
	Header: MessageHeader{
		Version:    10,
		Length:     420,
		ExportTime: 1483484756,
		SequenceNo: 2563920489,
		DomainID:   34560,
	},
	DataFlowSets: []DataFlowSet{
		{
			DataSets: [][]DecodedField{
				{
					{ID: 0x8, Value: net.IP{0x5b, 0x7d, 0x82, 0x79}},
					{ID: 0xc, Value: net.IP{0xc0, 0xe5, 0xdc, 0x85}},
					{ID: 0x5, Value: 0x0},
					{ID: 0x4, Value: 0x6},
					{ID: 0x7, Value: 0xecba},
					{ID: 0xb, Value: 0x1bb},
					{ID: 0x20, Value: 0x0},
					{ID: 0xa, Value: 0x503},
					{ID: 0x3a, Value: 0x0},
					{ID: 0x9, Value: 0x10},
					{ID: 0xd, Value: 0x18},
					{ID: 0x10, Value: 0x1ad7},
					{ID: 0x11, Value: 0x3b1d},
					{ID: 0xf, Value: net.IP{0xc0, 0x10, 0x1c, 0x58}},
					{ID: 0x6, Value: []uint8{0x10}},
					{ID: 0xe, Value: 0x4f6},
					{ID: 0x1, Value: 0x28},
					{ID: 0x2, Value: 0x1},
					{ID: 0x34, Value: 0x3a},
					{ID: 0x35, Value: 0x3a},
					{ID: 0x98, Value: 1483484685331},
					{ID: 0x99, Value: 1483484685331},
					{ID: 0x88, Value: 0x1},
					{ID: 0xf3, Value: 0x0},
					{ID: 0xf5, Value: 0x0},
				},
			},
		},
	},

}

func TestJSONMarshal(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := TestMessage{}

	b, err := mockDecodedMsg.JSONMarshal(buf,mockDecodedMsg.DataFlowSets[0].DataSets)
	if err != nil {
		t.Error("unexpected error", err)
	}

	err = json.Unmarshal(b, &msg)
	if err != nil {
		t.Error("unexpected error", err)
	}
	if msg.AgentID != "10.10.10.10" {
		t.Error("expect AgentID 10.10.10.10, got", msg.AgentID)
	}
	if msg.Header.Version != 10 {
		t.Error("expect Version 10, got", msg.Header.Version)
	}
}

func TestJSONMarshalDataSets(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := TestMessage{}

	b, _ := mockDecodedMsg.JSONMarshal(buf,mockDecodedMsg.DataFlowSets[0].DataSets)
	json.Unmarshal(b, &msg)

	for _, ds := range msg.DataSets {
		for _, f := range ds {
			switch f.I {
			case 1:
				chkFloat64(t, f, 40)
			case 2:
				chkFloat64(t, f, 1)
			case 4:
				chkFloat64(t, f, 6)
			case 5:
				chkFloat64(t, f, 0)
			case 6:
				chkString(t, f, "0x10")
			case 8:
				chkString(t, f, "91.125.130.121")
			case 12:
				chkString(t, f, "192.229.220.133")
			case 13:
				chkFloat64(t, f, 24)
			case 14:
				chkFloat64(t, f, 1270)
			case 152:
				chkFloat64(t, f, 1483484685331)
			}
		}
	}
}

func BenchmarkJSONMarshal(b *testing.B) {
	buf := new(bytes.Buffer)

	for i := 0; i < b.N; i++ {
		mockDecodedMsg.JSONMarshal(buf)
	}

}

func chkFloat64(t *testing.T, f TestDecodedField, expect float64) {
	if f.V.(float64) != expect {
		t.Errorf("expect ID %d value %f, got %f", f.I, expect, f.V)
	}
}

func chkString(t *testing.T, f TestDecodedField, expect string) {
	if f.V.(string) != expect {
		t.Errorf("expect ID %d value %s, got %s", f.I, expect, f.V.(string))
	}
}
