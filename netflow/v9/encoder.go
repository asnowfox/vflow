package netflow9

import (
	"encoding/binary"
	"bytes"

	"fmt"
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

func Encode(originalMsg Message, seq uint32, DataFlowSets []DataFlowSet) []byte {
	buf := new(bytes.Buffer)
	count := uint16(0)
	for _, e := range DataFlowSets {
		count += uint16(len(e.DataFlowRecords))
	}

	count = count + uint16(len(originalMsg.TemplateRecords))

	//orginal flow header
	binary.Write(buf, binary.BigEndian, originalMsg.Header.Version)
	binary.Write(buf, binary.BigEndian, uint16(count))
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SysUpTime)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.UNIXSecs)
	binary.Write(buf, binary.BigEndian, seq)
	binary.Write(buf, binary.BigEndian, originalMsg.Header.SrcID)

	for _, template := range originalMsg.TemplateRecords {
		fmt.Printf("write template record template id is %d, field count is %d.\n",
			template.Header.TemplateID, template.Header.FieldCount)
		writeTemplate(buf, template)
	}
	for _, flowSet := range DataFlowSets {
		binary.Write(buf, binary.BigEndian, flowSet.SetHeader.FlowSetID)
		binary.Write(buf, binary.BigEndian, flowSet.SetHeader.Length)
		for _, field := range flowSet.DataFlowRecords {
			for _, item := range field.DataSets {
				binary.Write(buf, binary.BigEndian, item.Value)
			}
		}
	}
	result := buf.Bytes()
	return result
}

func writeTemplate(buf *bytes.Buffer, TemplaRecord TemplateRecord) {
	if TemplaRecord.Header.FieldCount > 0 && TemplaRecord.SetId == 0 {
		binary.Write(buf, binary.BigEndian, uint16(TemplaRecord.SetId))
		binary.Write(buf, binary.BigEndian, uint16(4+4+4*TemplaRecord.Header.FieldCount))
		binary.Write(buf, binary.BigEndian, TemplaRecord.Header.TemplateID)
		binary.Write(buf, binary.BigEndian, TemplaRecord.Header.FieldCount)
		for _, spec := range TemplaRecord.FieldSpecifiers {
			binary.Write(buf, binary.BigEndian, spec.ElementID)
			binary.Write(buf, binary.BigEndian, spec.Length)
		}
	} else if TemplaRecord.SetId == 1{
		binary.Write(buf, binary.BigEndian, uint16(TemplaRecord.SetId))
		binary.Write(buf, binary.BigEndian, uint16(4 + 4 + 2 + 4*len(TemplaRecord.ScopeFieldSpecifiers) + 4*len(TemplaRecord.FieldSpecifiers))) //length
		binary.Write(buf, binary.BigEndian, TemplaRecord.Header.TemplateID)
		binary.Write(buf, binary.BigEndian, TemplaRecord.Header.OptionScopeLen)
		binary.Write(buf, binary.BigEndian, TemplaRecord.Header.OptionLen)
		for _,sfs := range TemplaRecord.ScopeFieldSpecifiers {
			binary.Write(buf,binary.BigEndian,sfs.ElementID)
			binary.Write(buf,binary.BigEndian,sfs.Length)
		}
		for _, spec := range TemplaRecord.FieldSpecifiers {
			binary.Write(buf, binary.BigEndian, spec.ElementID)
			binary.Write(buf, binary.BigEndian, spec.Length)
		}

	} else {
		fmt.Printf("template record's Field count is 0\n")
	}
}
