// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: pkg/types/snapshot/snapshot.proto

package snapshot

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	github_com_gogo_protobuf_types "github.com/gogo/protobuf/types"
	change "github.com/onosproject/onos-config/pkg/types/change"
	github_com_onosproject_onos_topo_pkg_northbound_device "github.com/onosproject/onos-topo/pkg/northbound/device"
	io "io"
	math "math"
	math_bits "math/bits"
	time "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// Status is the status of a configuration object
type Status int32

const (
	// PENDING indicates the configuration is waiting to be applied
	Status_PENDING Status = 0
	// APPLYING indicates the configuration is in the process of being applied
	Status_APPLYING Status = 1
	// SUCCEEDED indicates the configuration was applied successfully
	Status_SUCCEEDED Status = 2
	// FAILED indicates the configuration failed
	Status_FAILED Status = 3
)

var Status_name = map[int32]string{
	0: "PENDING",
	1: "APPLYING",
	2: "SUCCEEDED",
	3: "FAILED",
}

var Status_value = map[string]int32{
	"PENDING":   0,
	"APPLYING":  1,
	"SUCCEEDED": 2,
	"FAILED":    3,
}

func (x Status) String() string {
	return proto.EnumName(Status_name, int32(x))
}

func (Status) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_2795dc4c070a10ed, []int{0}
}

// Reason is a failure reason
type Reason int32

const (
	// ERROR indicates an error occurred
	Reason_ERROR Reason = 0
	// UNAVAILABLE indicates the device was unavailable
	Reason_UNAVAILABLE Reason = 1
)

var Reason_name = map[int32]string{
	0: "ERROR",
	1: "UNAVAILABLE",
}

var Reason_value = map[string]int32{
	"ERROR":       0,
	"UNAVAILABLE": 1,
}

func (x Reason) String() string {
	return proto.EnumName(Reason_name, int32(x))
}

func (Reason) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_2795dc4c070a10ed, []int{1}
}

// NetworkSnapshotRequest is a network snapshot request
type NetworkSnapshotRequest struct {
	// the request identifier
	ID ID `protobuf:"bytes,1,opt,name=id,proto3,casttype=ID" json:"id,omitempty"`
	// the request index
	Index Index `protobuf:"varint,2,opt,name=index,proto3,casttype=Index" json:"index,omitempty"`
	// status is the snapshot status
	Status Status `protobuf:"varint,3,opt,name=status,proto3,enum=onos.config.snapshot.Status" json:"status,omitempty"`
	// reason is the snapshot failure reason
	Reason Reason `protobuf:"varint,4,opt,name=reason,proto3,enum=onos.config.snapshot.Reason" json:"reason,omitempty"`
	// values is a list of values to set
	Devices []string `protobuf:"bytes,5,rep,name=devices,proto3" json:"devices,omitempty"`
	// timestamp is the time at which to take the snapshot
	Timestamp time.Time `protobuf:"bytes,6,opt,name=timestamp,proto3,stdtime" json:"timestamp"`
	// created is the time at which the configuration was created
	Created time.Time `protobuf:"bytes,7,opt,name=created,proto3,stdtime" json:"created"`
	// updated is the time at which the configuration was last updated
	Updated time.Time `protobuf:"bytes,8,opt,name=updated,proto3,stdtime" json:"updated"`
}

func (m *NetworkSnapshotRequest) Reset()         { *m = NetworkSnapshotRequest{} }
func (m *NetworkSnapshotRequest) String() string { return proto.CompactTextString(m) }
func (*NetworkSnapshotRequest) ProtoMessage()    {}
func (*NetworkSnapshotRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2795dc4c070a10ed, []int{0}
}
func (m *NetworkSnapshotRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *NetworkSnapshotRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_NetworkSnapshotRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *NetworkSnapshotRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NetworkSnapshotRequest.Merge(m, src)
}
func (m *NetworkSnapshotRequest) XXX_Size() int {
	return m.Size()
}
func (m *NetworkSnapshotRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_NetworkSnapshotRequest.DiscardUnknown(m)
}

var xxx_messageInfo_NetworkSnapshotRequest proto.InternalMessageInfo

func (m *NetworkSnapshotRequest) GetID() ID {
	if m != nil {
		return m.ID
	}
	return ""
}

func (m *NetworkSnapshotRequest) GetIndex() Index {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *NetworkSnapshotRequest) GetStatus() Status {
	if m != nil {
		return m.Status
	}
	return Status_PENDING
}

func (m *NetworkSnapshotRequest) GetReason() Reason {
	if m != nil {
		return m.Reason
	}
	return Reason_ERROR
}

func (m *NetworkSnapshotRequest) GetDevices() []string {
	if m != nil {
		return m.Devices
	}
	return nil
}

func (m *NetworkSnapshotRequest) GetTimestamp() time.Time {
	if m != nil {
		return m.Timestamp
	}
	return time.Time{}
}

func (m *NetworkSnapshotRequest) GetCreated() time.Time {
	if m != nil {
		return m.Created
	}
	return time.Time{}
}

func (m *NetworkSnapshotRequest) GetUpdated() time.Time {
	if m != nil {
		return m.Updated
	}
	return time.Time{}
}

// DeviceSnapshotRequest is a device snapshot request
type DeviceSnapshotRequest struct {
	// the request identifier
	ID ID `protobuf:"bytes,1,opt,name=id,proto3,casttype=ID" json:"id,omitempty"`
	// device_id is the device to which the snapshot applies
	DeviceID github_com_onosproject_onos_topo_pkg_northbound_device.ID `protobuf:"bytes,2,opt,name=device_id,json=deviceId,proto3,casttype=github.com/onosproject/onos-topo/pkg/northbound/device.ID" json:"device_id,omitempty"`
	// status is the snapshot status
	Status Status `protobuf:"varint,3,opt,name=status,proto3,enum=onos.config.snapshot.Status" json:"status,omitempty"`
	// reason is the snapshot failure reason
	Reason Reason `protobuf:"varint,4,opt,name=reason,proto3,enum=onos.config.snapshot.Reason" json:"reason,omitempty"`
	// timestamp is the time at which to take the snapshot
	Timestamp time.Time `protobuf:"bytes,5,opt,name=timestamp,proto3,stdtime" json:"timestamp"`
	// created is the time at which the configuration was created
	Created time.Time `protobuf:"bytes,6,opt,name=created,proto3,stdtime" json:"created"`
	// updated is the time at which the configuration was last updated
	Updated time.Time `protobuf:"bytes,7,opt,name=updated,proto3,stdtime" json:"updated"`
}

func (m *DeviceSnapshotRequest) Reset()         { *m = DeviceSnapshotRequest{} }
func (m *DeviceSnapshotRequest) String() string { return proto.CompactTextString(m) }
func (*DeviceSnapshotRequest) ProtoMessage()    {}
func (*DeviceSnapshotRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2795dc4c070a10ed, []int{1}
}
func (m *DeviceSnapshotRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DeviceSnapshotRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DeviceSnapshotRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DeviceSnapshotRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeviceSnapshotRequest.Merge(m, src)
}
func (m *DeviceSnapshotRequest) XXX_Size() int {
	return m.Size()
}
func (m *DeviceSnapshotRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeviceSnapshotRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeviceSnapshotRequest proto.InternalMessageInfo

func (m *DeviceSnapshotRequest) GetID() ID {
	if m != nil {
		return m.ID
	}
	return ""
}

func (m *DeviceSnapshotRequest) GetDeviceID() github_com_onosproject_onos_topo_pkg_northbound_device.ID {
	if m != nil {
		return m.DeviceID
	}
	return ""
}

func (m *DeviceSnapshotRequest) GetStatus() Status {
	if m != nil {
		return m.Status
	}
	return Status_PENDING
}

func (m *DeviceSnapshotRequest) GetReason() Reason {
	if m != nil {
		return m.Reason
	}
	return Reason_ERROR
}

func (m *DeviceSnapshotRequest) GetTimestamp() time.Time {
	if m != nil {
		return m.Timestamp
	}
	return time.Time{}
}

func (m *DeviceSnapshotRequest) GetCreated() time.Time {
	if m != nil {
		return m.Created
	}
	return time.Time{}
}

func (m *DeviceSnapshotRequest) GetUpdated() time.Time {
	if m != nil {
		return m.Updated
	}
	return time.Time{}
}

// DeviceSnapshot is a snapshot of a single device
type DeviceSnapshot struct {
	// values is a list of values to set
	Values []*change.Value `protobuf:"bytes,4,rep,name=values,proto3" json:"values,omitempty"`
	// timestamp is the time at which to take the snapshot
	Timestamp time.Time `protobuf:"bytes,5,opt,name=timestamp,proto3,stdtime" json:"timestamp"`
}

func (m *DeviceSnapshot) Reset()         { *m = DeviceSnapshot{} }
func (m *DeviceSnapshot) String() string { return proto.CompactTextString(m) }
func (*DeviceSnapshot) ProtoMessage()    {}
func (*DeviceSnapshot) Descriptor() ([]byte, []int) {
	return fileDescriptor_2795dc4c070a10ed, []int{2}
}
func (m *DeviceSnapshot) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DeviceSnapshot) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DeviceSnapshot.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DeviceSnapshot) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeviceSnapshot.Merge(m, src)
}
func (m *DeviceSnapshot) XXX_Size() int {
	return m.Size()
}
func (m *DeviceSnapshot) XXX_DiscardUnknown() {
	xxx_messageInfo_DeviceSnapshot.DiscardUnknown(m)
}

var xxx_messageInfo_DeviceSnapshot proto.InternalMessageInfo

func (m *DeviceSnapshot) GetValues() []*change.Value {
	if m != nil {
		return m.Values
	}
	return nil
}

func (m *DeviceSnapshot) GetTimestamp() time.Time {
	if m != nil {
		return m.Timestamp
	}
	return time.Time{}
}

func init() {
	proto.RegisterEnum("onos.config.snapshot.Status", Status_name, Status_value)
	proto.RegisterEnum("onos.config.snapshot.Reason", Reason_name, Reason_value)
	proto.RegisterType((*NetworkSnapshotRequest)(nil), "onos.config.snapshot.NetworkSnapshotRequest")
	proto.RegisterType((*DeviceSnapshotRequest)(nil), "onos.config.snapshot.DeviceSnapshotRequest")
	proto.RegisterType((*DeviceSnapshot)(nil), "onos.config.snapshot.DeviceSnapshot")
}

func init() { proto.RegisterFile("pkg/types/snapshot/snapshot.proto", fileDescriptor_2795dc4c070a10ed) }

var fileDescriptor_2795dc4c070a10ed = []byte{
	// 575 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xbc, 0x53, 0x41, 0x6f, 0x94, 0x4e,
	0x1c, 0x65, 0x96, 0x2e, 0xbb, 0xcc, 0xf6, 0xdf, 0xff, 0x66, 0x52, 0x0d, 0x36, 0x0d, 0x60, 0xe3,
	0x81, 0x34, 0x11, 0x62, 0xf5, 0xe2, 0xa5, 0x71, 0x29, 0x68, 0x48, 0x36, 0x6b, 0x33, 0xb5, 0x4d,
	0x3c, 0x19, 0x16, 0xa6, 0x2c, 0xb6, 0xcb, 0x20, 0x0c, 0x55, 0x3f, 0x81, 0xd7, 0x7e, 0x05, 0xbf,
	0x4d, 0x2f, 0x26, 0x3d, 0x7a, 0x5a, 0xcd, 0xee, 0xb7, 0xe8, 0xc9, 0x30, 0xb0, 0x6d, 0x36, 0x31,
	0x4d, 0xd6, 0x1a, 0x2f, 0xf0, 0x9b, 0x1f, 0xef, 0x3d, 0x7e, 0xf3, 0xde, 0x0c, 0x7c, 0x98, 0x9e,
	0x44, 0x16, 0xfb, 0x9c, 0x92, 0xdc, 0xca, 0x13, 0x3f, 0xcd, 0x47, 0x94, 0x5d, 0x17, 0x66, 0x9a,
	0x51, 0x46, 0xd1, 0x3a, 0x4d, 0x68, 0x6e, 0x06, 0x34, 0x39, 0x8e, 0x23, 0x73, 0xfe, 0x6d, 0x43,
	0x8b, 0x28, 0x8d, 0x4e, 0x89, 0xc5, 0x31, 0xc3, 0xe2, 0xd8, 0x62, 0xf1, 0x98, 0xe4, 0xcc, 0x1f,
	0xa7, 0x15, 0x6d, 0x63, 0x3d, 0xa2, 0x11, 0xe5, 0xa5, 0x55, 0x56, 0x75, 0xf7, 0x45, 0x14, 0xb3,
	0x51, 0x31, 0x34, 0x03, 0x3a, 0xb6, 0x4a, 0xdd, 0x34, 0xa3, 0xef, 0x49, 0xc0, 0x78, 0xfd, 0xb8,
	0xfa, 0x87, 0x75, 0x33, 0x52, 0x30, 0xf2, 0x93, 0x88, 0xd4, 0xaf, 0x4a, 0x61, 0xeb, 0xab, 0x08,
	0xef, 0x0f, 0x08, 0xfb, 0x48, 0xb3, 0x93, 0x83, 0x7a, 0x18, 0x4c, 0x3e, 0x14, 0x24, 0x67, 0x68,
	0x13, 0x36, 0xe2, 0x50, 0x01, 0x3a, 0x30, 0x64, 0x7b, 0x75, 0x3a, 0xd1, 0x1a, 0x9e, 0x73, 0xc5,
	0x9f, 0xb8, 0x11, 0x87, 0x48, 0x83, 0xcd, 0x38, 0x09, 0xc9, 0x27, 0xa5, 0xa1, 0x03, 0x63, 0xc5,
	0x96, 0xaf, 0x26, 0x5a, 0xd3, 0x2b, 0x1b, 0xb8, 0xea, 0xa3, 0x67, 0x50, 0xca, 0x99, 0xcf, 0x8a,
	0x5c, 0x11, 0x75, 0x60, 0xac, 0xed, 0x6c, 0x9a, 0xbf, 0xdb, 0xb9, 0x79, 0xc0, 0x31, 0xb8, 0xc6,
	0x96, 0xac, 0x8c, 0xf8, 0x39, 0x4d, 0x94, 0x95, 0xdb, 0x58, 0x98, 0x63, 0x70, 0x8d, 0x45, 0x0a,
	0x6c, 0x85, 0xe4, 0x2c, 0x0e, 0x48, 0xae, 0x34, 0x75, 0xd1, 0x90, 0xf1, 0x7c, 0x89, 0x6c, 0x28,
	0x5f, 0x5b, 0xa9, 0x48, 0x3a, 0x30, 0x3a, 0x3b, 0x1b, 0x66, 0x65, 0xb6, 0x39, 0x37, 0xdb, 0x7c,
	0x33, 0x47, 0xd8, 0xed, 0x8b, 0x89, 0x26, 0x9c, 0xff, 0xd0, 0x00, 0xbe, 0xa1, 0xa1, 0x5d, 0xd8,
	0x0a, 0x32, 0xe2, 0x33, 0x12, 0x2a, 0xad, 0x25, 0x14, 0xe6, 0xa4, 0x92, 0x5f, 0xa4, 0x21, 0xe7,
	0xb7, 0x97, 0xe1, 0xd7, 0xa4, 0xad, 0x6f, 0x22, 0xbc, 0xe7, 0xf0, 0xfd, 0x2c, 0x17, 0xd1, 0x31,
	0x94, 0x2b, 0x1b, 0xde, 0xc5, 0x21, 0x8f, 0x49, 0xb6, 0xbd, 0xe9, 0x44, 0x6b, 0x57, 0x5a, 0x1c,
	0xfa, 0xfc, 0xb6, 0x03, 0xc4, 0x68, 0x4a, 0xf9, 0xf1, 0x49, 0x68, 0xc6, 0x46, 0x43, 0x5a, 0x24,
	0xa1, 0x55, 0x09, 0x9a, 0x9e, 0x83, 0xdb, 0x55, 0xe9, 0x85, 0xff, 0x34, 0xe9, 0x85, 0x3c, 0x9b,
	0x77, 0xce, 0x53, 0xba, 0x63, 0x9e, 0xad, 0x3f, 0xc9, 0xf3, 0x0b, 0x80, 0x6b, 0x8b, 0x79, 0xa2,
	0x27, 0x50, 0x3a, 0xf3, 0x4f, 0x0b, 0x92, 0x2b, 0x2b, 0xba, 0x68, 0x74, 0x76, 0x1e, 0x2c, 0x98,
	0x51, 0xdf, 0xd8, 0xa3, 0x12, 0x81, 0x6b, 0xe0, 0xdf, 0x70, 0x62, 0x7b, 0x17, 0x4a, 0x55, 0x2a,
	0xa8, 0x03, 0x5b, 0xfb, 0xee, 0xc0, 0xf1, 0x06, 0xaf, 0xba, 0x02, 0x5a, 0x85, 0xed, 0xde, 0xfe,
	0x7e, 0xff, 0x6d, 0xb9, 0x02, 0xe8, 0x3f, 0x28, 0x1f, 0x1c, 0xee, 0xed, 0xb9, 0xae, 0xe3, 0x3a,
	0xdd, 0x06, 0x82, 0x50, 0x7a, 0xd9, 0xf3, 0xfa, 0xae, 0xd3, 0x15, 0xb7, 0x1f, 0x41, 0xa9, 0xca,
	0x07, 0xc9, 0xb0, 0xe9, 0x62, 0xfc, 0x1a, 0x77, 0x05, 0xf4, 0x3f, 0xec, 0x1c, 0x0e, 0x7a, 0x47,
	0x3d, 0xaf, 0xdf, 0xb3, 0xfb, 0x6e, 0x17, 0xd8, 0xca, 0xc5, 0x54, 0x05, 0x97, 0x53, 0x15, 0xfc,
	0x9c, 0xaa, 0xe0, 0x7c, 0xa6, 0x0a, 0x97, 0x33, 0x55, 0xf8, 0x3e, 0x53, 0x85, 0xa1, 0xc4, 0x07,
	0x7d, 0xfa, 0x2b, 0x00, 0x00, 0xff, 0xff, 0x27, 0xe6, 0x7c, 0xb3, 0x38, 0x05, 0x00, 0x00,
}

func (m *NetworkSnapshotRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *NetworkSnapshotRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *NetworkSnapshotRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n1, err1 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Updated, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Updated):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintSnapshot(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x42
	n2, err2 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Created, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Created):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintSnapshot(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x3a
	n3, err3 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Timestamp, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Timestamp):])
	if err3 != nil {
		return 0, err3
	}
	i -= n3
	i = encodeVarintSnapshot(dAtA, i, uint64(n3))
	i--
	dAtA[i] = 0x32
	if len(m.Devices) > 0 {
		for iNdEx := len(m.Devices) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Devices[iNdEx])
			copy(dAtA[i:], m.Devices[iNdEx])
			i = encodeVarintSnapshot(dAtA, i, uint64(len(m.Devices[iNdEx])))
			i--
			dAtA[i] = 0x2a
		}
	}
	if m.Reason != 0 {
		i = encodeVarintSnapshot(dAtA, i, uint64(m.Reason))
		i--
		dAtA[i] = 0x20
	}
	if m.Status != 0 {
		i = encodeVarintSnapshot(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x18
	}
	if m.Index != 0 {
		i = encodeVarintSnapshot(dAtA, i, uint64(m.Index))
		i--
		dAtA[i] = 0x10
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.ID)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *DeviceSnapshotRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DeviceSnapshotRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DeviceSnapshotRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n4, err4 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Updated, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Updated):])
	if err4 != nil {
		return 0, err4
	}
	i -= n4
	i = encodeVarintSnapshot(dAtA, i, uint64(n4))
	i--
	dAtA[i] = 0x3a
	n5, err5 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Created, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Created):])
	if err5 != nil {
		return 0, err5
	}
	i -= n5
	i = encodeVarintSnapshot(dAtA, i, uint64(n5))
	i--
	dAtA[i] = 0x32
	n6, err6 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Timestamp, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Timestamp):])
	if err6 != nil {
		return 0, err6
	}
	i -= n6
	i = encodeVarintSnapshot(dAtA, i, uint64(n6))
	i--
	dAtA[i] = 0x2a
	if m.Reason != 0 {
		i = encodeVarintSnapshot(dAtA, i, uint64(m.Reason))
		i--
		dAtA[i] = 0x20
	}
	if m.Status != 0 {
		i = encodeVarintSnapshot(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x18
	}
	if len(m.DeviceID) > 0 {
		i -= len(m.DeviceID)
		copy(dAtA[i:], m.DeviceID)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.DeviceID)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintSnapshot(dAtA, i, uint64(len(m.ID)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *DeviceSnapshot) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DeviceSnapshot) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DeviceSnapshot) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n7, err7 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Timestamp, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Timestamp):])
	if err7 != nil {
		return 0, err7
	}
	i -= n7
	i = encodeVarintSnapshot(dAtA, i, uint64(n7))
	i--
	dAtA[i] = 0x2a
	if len(m.Values) > 0 {
		for iNdEx := len(m.Values) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Values[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintSnapshot(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintSnapshot(dAtA []byte, offset int, v uint64) int {
	offset -= sovSnapshot(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *NetworkSnapshotRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	if m.Index != 0 {
		n += 1 + sovSnapshot(uint64(m.Index))
	}
	if m.Status != 0 {
		n += 1 + sovSnapshot(uint64(m.Status))
	}
	if m.Reason != 0 {
		n += 1 + sovSnapshot(uint64(m.Reason))
	}
	if len(m.Devices) > 0 {
		for _, s := range m.Devices {
			l = len(s)
			n += 1 + l + sovSnapshot(uint64(l))
		}
	}
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Timestamp)
	n += 1 + l + sovSnapshot(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Created)
	n += 1 + l + sovSnapshot(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Updated)
	n += 1 + l + sovSnapshot(uint64(l))
	return n
}

func (m *DeviceSnapshotRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	l = len(m.DeviceID)
	if l > 0 {
		n += 1 + l + sovSnapshot(uint64(l))
	}
	if m.Status != 0 {
		n += 1 + sovSnapshot(uint64(m.Status))
	}
	if m.Reason != 0 {
		n += 1 + sovSnapshot(uint64(m.Reason))
	}
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Timestamp)
	n += 1 + l + sovSnapshot(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Created)
	n += 1 + l + sovSnapshot(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Updated)
	n += 1 + l + sovSnapshot(uint64(l))
	return n
}

func (m *DeviceSnapshot) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Values) > 0 {
		for _, e := range m.Values {
			l = e.Size()
			n += 1 + l + sovSnapshot(uint64(l))
		}
	}
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Timestamp)
	n += 1 + l + sovSnapshot(uint64(l))
	return n
}

func sovSnapshot(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozSnapshot(x uint64) (n int) {
	return sovSnapshot(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *NetworkSnapshotRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnapshot
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: NetworkSnapshotRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: NetworkSnapshotRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ID = ID(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Index", wireType)
			}
			m.Index = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Index |= Index(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= Status(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reason", wireType)
			}
			m.Reason = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Reason |= Reason(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Devices", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Devices = append(m.Devices, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Timestamp, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Created", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Created, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Updated", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Updated, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSnapshot(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DeviceSnapshotRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnapshot
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DeviceSnapshotRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DeviceSnapshotRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ID = ID(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DeviceID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DeviceID = github_com_onosproject_onos_topo_pkg_northbound_device.ID(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= Status(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reason", wireType)
			}
			m.Reason = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Reason |= Reason(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Timestamp, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Created", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Created, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Updated", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Updated, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSnapshot(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *DeviceSnapshot) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSnapshot
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: DeviceSnapshot: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DeviceSnapshot: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Values", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Values = append(m.Values, &change.Value{})
			if err := m.Values[len(m.Values)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSnapshot
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSnapshot
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Timestamp, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSnapshot(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthSnapshot
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipSnapshot(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSnapshot
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSnapshot
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthSnapshot
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthSnapshot
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowSnapshot
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipSnapshot(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthSnapshot
				}
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthSnapshot = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSnapshot   = fmt.Errorf("proto: integer overflow")
)
