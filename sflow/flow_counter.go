//: ----------------------------------------------------------------------------
//: Copyright (C) 2017 Verizon.  All Rights Reserved.
//: All Rights Reserved
//:
//: file:    flow_counter.go
//: details: TODO
//: author:  Mehrdad Arshad Rad
//: date:    08/08/2018
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

package sflow

import "io"

const (
	// SFGenericInterfaceCounters is Generic interface counters - see RFC 2233
	SFGenericInterfaceCounters = 1

	// SFEthernetInterfaceCounters is Ethernet interface counters - see RFC 2358
	SFEthernetInterfaceCounters = 2

	// SFTokenRingInterfaceCounters is Token ring counters - see RFC 1748
	SFTokenRingInterfaceCounters = 3

	// SF100BaseVGInterfaceCounters is 100 BaseVG interface counters - see RFC 2020
	SF100BaseVGInterfaceCounters = 4

	// SFVLANCounters is VLAN counters
	SFVLANCounters = 5

	// SFProcessorCounters is processor counters
	SFProcessorCounters = 1001
)

// GenericInterfaceCounters represents Generic Interface Counters RFC2233
type GenericInterfaceCounters struct {
	Index               uint32 `json:"index"`
	Type                uint32 `json:"type"`
	Speed               uint64 `json:"speed"`
	Direction           uint32 `json:"direction"`
	Status              uint32 `json:"status"`
	InOctets            uint64 `json:"in_octets"`
	InUnicastPackets    uint32 `json:"in_unicast_packets"`
	InMulticastPackets  uint32 `json:"in_multicast_packets"`
	InBroadcastPackets  uint32 `json:"in_broadcast_packets"`
	InDiscards          uint32 `json:"in_discards"`
	InErrors            uint32 `json:"in_errors"`
	InUnknownProtocols  uint32 `json:"in_unknown_protocols"`
	OutOctets           uint64 `json:"out_octets"`
	OutUnicastPackets   uint32 `json:"out_unicast_packets"`
	OutMulticastPackets uint32 `json:"out_multicast_packets"`
	OutBroadcastPackets uint32 `json:"out_broadcast_packets"`
	OutDiscards         uint32 `json:"out_discards"`
	OutErrors           uint32 `json:"out_errors"`
	PromiscuousMode     uint32 `json:"promiscuous_mode"`
}

// EthernetInterfaceCounters represents Ethernet Interface Counters RFC2358
type EthernetInterfaceCounters struct {
	AlignmentErrors           uint32 `json:"alignment_errors"`
	FCSErrors                 uint32 `json:"fca_errors "`
	SingleCollisionFrames     uint32 `json:"single_collision_frames"`
	MultipleCollisionFrames   uint32 `json:"multiple_collision_frames"`
	SQETestErrors             uint32 `json:"sqe_test_errors"`
	DeferredTransmissions     uint32 `json:"deferred_transmissions"`
	LateCollisions            uint32 `json:"late_collisions"`
	ExcessiveCollisions       uint32 `json:"excessive_collisions"`
	InternalMACTransmitErrors uint32 `json:"internal_mac_transmit_errors"`
	CarrierSenseErrors        uint32 `json:"carrier_sense_errors"`
	FrameTooLongs             uint32 `json:"frame_too_longs"`
	InternalMACReceiveErrors  uint32 `json:"internal_mac_receive_errors"`
	SymbolErrors              uint32 `json:"symbol_errors"`
}

// TokenRingCounters represents Token Ring Counters - see RFC 1748
type TokenRingCounters struct {
	LineErrors         uint32 `json:"index"`
	BurstErrors        uint32 `json:"index"`
	ACErrors           uint32 `json:"index"`
	AbortTransErrors   uint32 `json:"index"`
	InternalErrors     uint32 `json:"index"`
	LostFrameErrors    uint32 `json:"index"`
	ReceiveCongestions uint32 `json:"index"`
	FrameCopiedErrors  uint32 `json:"index"`
	TokenErrors        uint32 `json:"index"`
	SoftErrors         uint32 `json:"index"`
	HardErrors         uint32 `json:"index"`
	SignalLoss         uint32 `json:"index"`
	TransmitBeacons    uint32 `json:"index"`
	Recoverys          uint32 `json:"index"`
	LobeWires          uint32 `json:"index"`
	Removes            uint32 `json:"index"`
	Singles            uint32 `json:"index"`
	FreqErrors         uint32 `json:"index"`
}

// VGCounters represents 100 BaseVG interface counters - see RFC 2020
type VGCounters struct {
	InHighPriorityFrames    uint32 `json:"in_high_priority_frames"`
	InHighPriorityOctets    uint64 `json:"in_high_priority_octets"`
	InNormPriorityFrames    uint32 `json:"in_norm_priority_frames"`
	InNormPriorityOctets    uint64 `json:"in_norm_priority_octets"`
	InIPMErrors             uint32 `json:"in_ipm_errors"`
	InOversizeFrameErrors   uint32 `json:"in_oversize_frame_errors"`
	InDataErrors            uint32 `json:"in_data_errors"`
	InNullAddressedFrames   uint32 `json:"in_null_addressed_frames"`
	OutHighPriorityFrames   uint32 `json:"out_high_priority_frames"`
	OutHighPriorityOctets   uint64 `json:"out_high_priority_octets"`
	TransitionIntoTrainings uint32 `json:"transition_into_trainings"`
	HCInHighPriorityOctets  uint64 `json:"hc_in_high_priority_octets"`
	HCInNormPriorityOctets  uint64 `json:"hc_in_norm_priority_octets"`
	HCOutHighPriorityOctets uint64 `json:"hc_out_high_priority_octets"`
}

// VlanCounters represents VLAN Counters
type VlanCounters struct {
	ID               uint32 `json:"id"`
	Octets           uint64 `json:"octets"`
	UnicastPackets   uint32 `json:"unicast_packets"`
	MulticastPackets uint32 `json:"multicast_packets"`
	BroadcastPackets uint32 `json:"broadcast_packets"`
	Discards         uint32 `json:"discards"`
}

// ProcessorCounters represents Processor Information
type ProcessorCounters struct {
	CPU5s       uint32 `json:"cpu5s"`
	CPU1m       uint32 `json:"cpu1m"`
	CPU5m       uint32 `json:"cpu5m"`
	TotalMemory uint64 `json:"total_memory"`
	FreeMemory  uint64 `json:"free_memory"`
}

// CounterSample represents the periodic sampling or polling of counters associated with a Data Source
type CounterSample struct {
	SequenceNo   uint32            `json:"sequence_no"`
	SourceIDType byte              `json:"source_id_type"`
	SourceIDIdx  uint32            `json:"source_id_idx"`
	RecordsNo    uint32            `json:"records_no"`
	Records      map[string]Record `json:"records"`
}

func decodeFlowCounter(r io.ReadSeeker) (*CounterSample, error) {
	var (
		cs          = new(CounterSample)
		rTypeFormat uint32
		rTypeLength uint32
		err         error
	)

	if err = cs.unmarshal(r); err != nil {
		return nil, err
	}

	cs.Records = make(map[string]Record)

	for i := uint32(0); i < cs.RecordsNo; i++ {
		if err = read(r, &rTypeFormat); err != nil {
			return nil, err
		}
		if err = read(r, &rTypeLength); err != nil {
			return nil, err
		}

		switch rTypeFormat {

		case SFGenericInterfaceCounters:
			d, err := decodeGenericIntCounters(r)
			if err != nil {
				return cs, err
			}
			cs.Records["GenInt"] = d
		case SFEthernetInterfaceCounters:
			d, err := decodeEthIntCounters(r)
			if err != nil {
				return cs, err
			}
			cs.Records["EthInt"] = d
		case SFTokenRingInterfaceCounters:
			d, err := decodeTokenRingCounters(r)
			if err != nil {
				return cs, err
			}
			cs.Records["TRInt"] = d
		case SF100BaseVGInterfaceCounters:
			d, err := decodeVGCounters(r)
			if err != nil {
				return cs, err
			}
			cs.Records["VGInt"] = d
		case SFVLANCounters:
			d, err := decodeVlanCounters(r)
			if err != nil {
				return cs, err
			}
			cs.Records["Vlan"] = d
		case SFProcessorCounters:
			d, err := decodedProcessorCounters(r)
			if err != nil {
				return cs, err
			}
			cs.Records["Proc"] = d
		default:
			r.Seek(int64(rTypeLength), 1)
		}
	}

	return cs, nil
}

func decodeGenericIntCounters(r io.Reader) (*GenericInterfaceCounters, error) {
	var gic = new(GenericInterfaceCounters)

	if err := gic.unmarshal(r); err != nil {
		return nil, err
	}

	return gic, nil
}

func (gic *GenericInterfaceCounters) unmarshal(r io.Reader) error {
	var err error

	fields := []interface{}{
		&gic.Index,
		&gic.Type,
		&gic.Speed,
		&gic.Direction,
		&gic.Status,
		&gic.InOctets,
		&gic.InUnicastPackets,
		&gic.InMulticastPackets,
		&gic.InBroadcastPackets,
		&gic.InDiscards,
		&gic.InErrors,
		&gic.InUnknownProtocols,
		&gic.OutOctets,
		&gic.OutUnicastPackets,
		&gic.OutMulticastPackets,
		&gic.OutBroadcastPackets,
		&gic.OutDiscards,
		&gic.OutErrors,
		&gic.PromiscuousMode,
	}

	for _, field := range fields {
		if err = read(r, field); err != nil {
			return err
		}
	}

	return nil
}
func decodeEthIntCounters(r io.Reader) (*EthernetInterfaceCounters, error) {
	var eic = new(EthernetInterfaceCounters)

	if err := eic.unmarshal(r); err != nil {
		return nil, err
	}

	return eic, nil
}

func (eic *EthernetInterfaceCounters) unmarshal(r io.Reader) error {
	var err error

	fields := []interface{}{
		&eic.AlignmentErrors,
		&eic.FCSErrors,
		&eic.SingleCollisionFrames,
		&eic.MultipleCollisionFrames,
		&eic.SQETestErrors,
		&eic.DeferredTransmissions,
		&eic.LateCollisions,
		&eic.ExcessiveCollisions,
		&eic.InternalMACTransmitErrors,
		&eic.CarrierSenseErrors,
		&eic.FrameTooLongs,
		&eic.InternalMACReceiveErrors,
		&eic.SymbolErrors,
	}

	for _, field := range fields {
		if err = read(r, field); err != nil {
			return err
		}
	}

	return nil
}
func decodeTokenRingCounters(r io.Reader) (*TokenRingCounters, error) {
	var tr = new(TokenRingCounters)

	if err := tr.unmarshal(r); err != nil {
		return nil, err
	}

	return tr, nil
}

func (tr *TokenRingCounters) unmarshal(r io.Reader) error {
	var err error

	fields := []interface{}{
		&tr.LineErrors,
		&tr.BurstErrors,
		&tr.ACErrors,
		&tr.AbortTransErrors,
		&tr.InternalErrors,
		&tr.LostFrameErrors,
		&tr.ReceiveCongestions,
		&tr.FrameCopiedErrors,
		&tr.TokenErrors,
		&tr.SoftErrors,
		&tr.HardErrors,
		&tr.SignalLoss,
		&tr.TransmitBeacons,
		&tr.Recoverys,
		&tr.LobeWires,
		&tr.Removes,
		&tr.Singles,
		&tr.FreqErrors,
	}

	for _, field := range fields {
		if err = read(r, field); err != nil {
			return err
		}
	}

	return nil
}

func decodeVGCounters(r io.Reader) (*VGCounters, error) {
	var vg = new(VGCounters)

	if err := vg.unmarshal(r); err != nil {
		return nil, err
	}

	return vg, nil
}

func (vg *VGCounters) unmarshal(r io.Reader) error {
	var err error

	fields := []interface{}{
		&vg.InHighPriorityFrames,
		&vg.InHighPriorityOctets,
		&vg.InNormPriorityFrames,
		&vg.InNormPriorityOctets,
		&vg.InIPMErrors,
		&vg.InOversizeFrameErrors,
		&vg.InDataErrors,
		&vg.InNullAddressedFrames,
		&vg.OutHighPriorityFrames,
		&vg.OutHighPriorityOctets,
		&vg.TransitionIntoTrainings,
		&vg.HCInHighPriorityOctets,
		&vg.HCInNormPriorityOctets,
		&vg.HCOutHighPriorityOctets,
	}

	for _, field := range fields {
		if err = read(r, field); err != nil {
			return err
		}
	}

	return nil
}

func decodeVlanCounters(r io.Reader) (*VlanCounters, error) {
	var vc = new(VlanCounters)

	if err := vc.unmarshal(r); err != nil {
		return nil, err
	}

	return vc, nil
}

func (vc *VlanCounters) unmarshal(r io.Reader) error {
	var err error
	fields := []interface{}{
		&vc.ID,
		&vc.Octets,
		&vc.UnicastPackets,
		&vc.MulticastPackets,
		&vc.BroadcastPackets,
		&vc.Discards,
	}

	for _, field := range fields {
		if err = read(r, field); err != nil {
			return err
		}
	}

	return nil
}

func decodedProcessorCounters(r io.Reader) (*ProcessorCounters, error) {
	var pc = new(ProcessorCounters)

	if err := pc.unmarshal(r); err != nil {
		return nil, err
	}

	return pc, nil
}

func (pc *ProcessorCounters) unmarshal(r io.Reader) error {
	var err error
	fields := []interface{}{
		&pc.CPU5s,
		&pc.CPU1m,
		&pc.CPU5m,
		&pc.TotalMemory,
		&pc.FreeMemory,
	}

	for _, field := range fields {
		if err = read(r, field); err != nil {
			return err
		}
	}

	return nil
}

func (cs *CounterSample) unmarshal(r io.Reader) error {

	var err error

	if err = read(r, &cs.SequenceNo); err != nil {
		return err
	}

	if err = read(r, &cs.SourceIDType); err != nil {
		return err
	}

	buf := make([]byte, 3)
	if err = read(r, &buf); err != nil {
		return err
	}
	cs.SourceIDIdx = uint32(buf[2]) | uint32(buf[1])<<8 | uint32(buf[0])<<16

	err = read(r, &cs.RecordsNo)

	return err
}
