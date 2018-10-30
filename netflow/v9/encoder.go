package netflow9

import (
	"encoding/binary"
	"bytes"

)

//   The Packet Header format is specified as:
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |       Version Number          |            Count              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                           sysUpTime                           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                           UNIX Secs                           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                       Sequence Number                         |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                        Source ID                              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// template header 信息
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      FlowSet ID  = 0          |          Length               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// template 描述信息
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      Template ID 256          |         Field Count           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Field Type 1           |         Field Length 1        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Field Type 2           |         Field Length 2        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |             ...               |              ...              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Field Type N           |         Field Length N        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// data 头部信息
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |      FlowSet ID  = 256        |          Length               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// data具体信息
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |        Field Type             |         Field Length          |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

func  Encode(originalMsg Message, seq uint32, rHeader SetHeader, flowSets  []FlowSet) []byte {
	buf := new(bytes.Buffer)
	count := uint16(0)
	for _,e := range flowSets {
		count += uint16(len(e.DataSets))
	}

	count = count + uint16(len(originalMsg.TemplateRecords))

	//orginal flow header
	binary.Write(buf, binary.BigEndian, originalMsg.Header.Version)
	binary.Write(buf, binary.BigEndian, uint16(count))
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SysUpTime)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.UNIXSecs)
	binary.Write(buf, binary.BigEndian, seq)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SrcID)

	for _,template := range originalMsg.TemplateRecords {
		writeTemplate(buf,template)
	}
	for _,flowSet := range flowSets {
		binary.Write(buf, binary.BigEndian, rHeader.FlowSetID)
		binary.Write(buf, binary.BigEndian, rHeader.Length)
		for _, field := range flowSet.DataSets {
			for _, item := range field {
				binary.Write(buf, binary.BigEndian, item.Value)
			}
		}
	}
	result := buf.Bytes()
	return result
}

func writeTemplate(buf *bytes.Buffer,TemplaRecord TemplateRecord){
	if TemplaRecord.FieldCount > 0 {
		binary.Write(buf, binary.BigEndian, 0)
		binary.Write(buf, binary.BigEndian, 4 + 4 + 4*TemplaRecord.FieldCount)

		binary.Write(buf, binary.BigEndian, TemplaRecord.TemplateID)
		binary.Write(buf, binary.BigEndian,TemplaRecord.FieldCount)
		for _, spec := range TemplaRecord.FieldSpecifiers {
			binary.Write(buf, binary.BigEndian, spec.ElementID)
			binary.Write(buf, binary.BigEndian, spec.Length)
		}
		if TemplaRecord.ScopeFieldCount > 0 {
			for _, spec1 := range TemplaRecord.ScopeFieldSpecifiers {
				binary.Write(buf, binary.BigEndian, spec1.ElementID)
				binary.Write(buf, binary.BigEndian, spec1.Length)
			}
		}
	}
}
