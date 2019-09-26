package sflow

import "io"

//ExpandedFlowSample represents expanded flow sample
type ExpandedFlowSample struct {
	SequenceNo   uint32 // Incremented with each flow sample
	SourceID     byte   // sfSourceID
	SamplingRate uint32 // sfPacketSamplingRate
	SamplePool   uint32 // Total number of packets that could have been sampled
	Drops        uint32 // Number of times a packet was dropped due to lack of resources
	InputFormat  uint32
	Input        uint32 // SNMP ifIndex of input interface
	OutputFormat uint32
	Output       uint32 // SNMP ifIndex of input interface
	RecordsNo    uint32 // Number of records to follow
	Records      map[string]Record
}

func decodeExpandedFlowSample(r io.ReadSeeker) (*ExpandedFlowSample, error) {
	var (
		efs         = new(ExpandedFlowSample)
		rTypeFormat uint32
		rTypeLength uint32
		err         error
	)

	if err = efs.unmarshal(r); err != nil {
		return nil, err
	}

	efs.Records = make(map[string]Record)

	for i := uint32(0); i < efs.RecordsNo; i++ {
		if err = read(r, &rTypeFormat); err != nil {
			return nil, err
		}
		if err = read(r, &rTypeLength); err != nil {
			return nil, err
		}

		switch rTypeFormat {
		case SFDataRawHeader:
			d, err := decodeSampledHeader(r)
			if err != nil {
				return efs, err
			}
			efs.Records["RawHeader"] = d
		case SFDataExtSwitch:
			d, err := decodeExtSwitchData(r)
			if err != nil {
				return efs, err
			}

			efs.Records["ExtSwitch"] = d
		case SFDataExtRouter:
			d, err := decodeExtRouterData(r, rTypeLength)
			if err != nil {
				return efs, err
			}

			efs.Records["ExtRouter"] = d
		default:
			r.Seek(int64(rTypeLength), 1)
		}
	}

	return efs, nil
}

func (fs *ExpandedFlowSample) unmarshal(r io.ReadSeeker) error {
	var err error

	if err = read(r, &fs.SequenceNo); err != nil {
		return err
	}

	if err = read(r, &fs.SourceID); err != nil {
		return err
	}

	r.Seek(3, 1) // skip counter sample decoding

	if err = read(r, &fs.SamplingRate); err != nil {
		return err
	}

	if err = read(r, &fs.SamplePool); err != nil {
		return err
	}

	if err = read(r, &fs.Drops); err != nil {
		return err
	}

	if err = read(r, &fs.InputFormat); err != nil {
		return err
	}

	if err = read(r, &fs.Input); err != nil {
		return err
	}

	if err = read(r, &fs.OutputFormat); err != nil {
		return err
	}

	if err = read(r, &fs.Output); err != nil {
		return err
	}

	err = read(r, &fs.RecordsNo)

	return err
}
