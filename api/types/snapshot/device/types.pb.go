// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: api/types/snapshot/device/types.proto

package device

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	github_com_gogo_protobuf_types "github.com/gogo/protobuf/types"
	github_com_onosproject_onos_config_api_types "github.com/onosproject/onos-config/api/types"
	device "github.com/onosproject/onos-config/api/types/change/device"
	github_com_onosproject_onos_config_api_types_change_device "github.com/onosproject/onos-config/api/types/change/device"
	github_com_onosproject_onos_config_api_types_device "github.com/onosproject/onos-config/api/types/device"
	snapshot "github.com/onosproject/onos-config/api/types/snapshot"
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
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// DeviceSnapshot is a device snapshot
type DeviceSnapshot struct {
	// 'id' is the unique snapshot identifier
	ID ID `protobuf:"bytes,1,opt,name=id,proto3,casttype=ID" json:"id,omitempty"`
	// 'device_id' is the device to which the snapshot applies
	DeviceID github_com_onosproject_onos_config_api_types_device.ID `protobuf:"bytes,2,opt,name=device_id,json=deviceId,proto3,casttype=github.com/onosproject/onos-config/api/types/device.ID" json:"device_id,omitempty"`
	// 'device_version' is the version to which the snapshot applies
	DeviceVersion github_com_onosproject_onos_config_api_types_device.Version `protobuf:"bytes,3,opt,name=device_version,json=deviceVersion,proto3,casttype=github.com/onosproject/onos-config/api/types/device.Version" json:"device_version,omitempty"`
	// 'revision' is the request revision number
	Revision Revision `protobuf:"varint,4,opt,name=revision,proto3,casttype=Revision" json:"revision,omitempty"`
	// 'network_snapshot' is a reference to the network snapshot from which this snapshot was created
	NetworkSnapshot NetworkSnapshotRef `protobuf:"bytes,5,opt,name=network_snapshot,json=networkSnapshot,proto3" json:"network_snapshot"`
	// 'max_network_change_index' is the maximum network change index to be snapshotted for the device
	MaxNetworkChangeIndex github_com_onosproject_onos_config_api_types.Index `protobuf:"varint,6,opt,name=max_network_change_index,json=maxNetworkChangeIndex,proto3,casttype=github.com/onosproject/onos-config/api/types.Index" json:"max_network_change_index,omitempty"`
	// 'status' is the snapshot status
	Status snapshot.Status `protobuf:"bytes,7,opt,name=status,proto3" json:"status"`
	// 'created' is the time at which the configuration was created
	Created time.Time `protobuf:"bytes,8,opt,name=created,proto3,stdtime" json:"created"`
	// 'updated' is the time at which the configuration was last updated
	Updated time.Time `protobuf:"bytes,9,opt,name=updated,proto3,stdtime" json:"updated"`
}

func (m *DeviceSnapshot) Reset()         { *m = DeviceSnapshot{} }
func (m *DeviceSnapshot) String() string { return proto.CompactTextString(m) }
func (*DeviceSnapshot) ProtoMessage()    {}
func (*DeviceSnapshot) Descriptor() ([]byte, []int) {
	return fileDescriptor_0f739fc21bf8c083, []int{0}
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

func (m *DeviceSnapshot) GetID() ID {
	if m != nil {
		return m.ID
	}
	return ""
}

func (m *DeviceSnapshot) GetDeviceID() github_com_onosproject_onos_config_api_types_device.ID {
	if m != nil {
		return m.DeviceID
	}
	return ""
}

func (m *DeviceSnapshot) GetDeviceVersion() github_com_onosproject_onos_config_api_types_device.Version {
	if m != nil {
		return m.DeviceVersion
	}
	return ""
}

func (m *DeviceSnapshot) GetRevision() Revision {
	if m != nil {
		return m.Revision
	}
	return 0
}

func (m *DeviceSnapshot) GetNetworkSnapshot() NetworkSnapshotRef {
	if m != nil {
		return m.NetworkSnapshot
	}
	return NetworkSnapshotRef{}
}

func (m *DeviceSnapshot) GetMaxNetworkChangeIndex() github_com_onosproject_onos_config_api_types.Index {
	if m != nil {
		return m.MaxNetworkChangeIndex
	}
	return 0
}

func (m *DeviceSnapshot) GetStatus() snapshot.Status {
	if m != nil {
		return m.Status
	}
	return snapshot.Status{}
}

func (m *DeviceSnapshot) GetCreated() time.Time {
	if m != nil {
		return m.Created
	}
	return time.Time{}
}

func (m *DeviceSnapshot) GetUpdated() time.Time {
	if m != nil {
		return m.Updated
	}
	return time.Time{}
}

// NetworkSnapshotRef is a back reference to the NetworkSnapshot that created a DeviceSnapshot
type NetworkSnapshotRef struct {
	// 'id' is the identifier of the network snapshot from which this snapshot was created
	ID github_com_onosproject_onos_config_api_types.ID `protobuf:"bytes,1,opt,name=id,proto3,casttype=github.com/onosproject/onos-config/api/types.ID" json:"id,omitempty"`
	// 'index' is the index of the network snapshot from which this snapshot was created
	Index github_com_onosproject_onos_config_api_types.Index `protobuf:"varint,2,opt,name=index,proto3,casttype=github.com/onosproject/onos-config/api/types.Index" json:"index,omitempty"`
}

func (m *NetworkSnapshotRef) Reset()         { *m = NetworkSnapshotRef{} }
func (m *NetworkSnapshotRef) String() string { return proto.CompactTextString(m) }
func (*NetworkSnapshotRef) ProtoMessage()    {}
func (*NetworkSnapshotRef) Descriptor() ([]byte, []int) {
	return fileDescriptor_0f739fc21bf8c083, []int{1}
}
func (m *NetworkSnapshotRef) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *NetworkSnapshotRef) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_NetworkSnapshotRef.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *NetworkSnapshotRef) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NetworkSnapshotRef.Merge(m, src)
}
func (m *NetworkSnapshotRef) XXX_Size() int {
	return m.Size()
}
func (m *NetworkSnapshotRef) XXX_DiscardUnknown() {
	xxx_messageInfo_NetworkSnapshotRef.DiscardUnknown(m)
}

var xxx_messageInfo_NetworkSnapshotRef proto.InternalMessageInfo

func (m *NetworkSnapshotRef) GetID() github_com_onosproject_onos_config_api_types.ID {
	if m != nil {
		return m.ID
	}
	return ""
}

func (m *NetworkSnapshotRef) GetIndex() github_com_onosproject_onos_config_api_types.Index {
	if m != nil {
		return m.Index
	}
	return 0
}

// Snapshot is a snapshot of the state of a single device
type Snapshot struct {
	// 'id' is a unique snapshot identifier
	ID ID `protobuf:"bytes,1,opt,name=id,proto3,casttype=ID" json:"id,omitempty"`
	// 'device_id' is the device to which the snapshot applies
	DeviceID github_com_onosproject_onos_config_api_types_device.ID `protobuf:"bytes,2,opt,name=device_id,json=deviceId,proto3,casttype=github.com/onosproject/onos-config/api/types/device.ID" json:"device_id,omitempty"`
	// 'device_version' is the version to which the snapshot applies
	DeviceVersion github_com_onosproject_onos_config_api_types_device.Version `protobuf:"bytes,3,opt,name=device_version,json=deviceVersion,proto3,casttype=github.com/onosproject/onos-config/api/types/device.Version" json:"device_version,omitempty"`
	// 'snapshot_id' is the ID of the snapshot
	SnapshotID ID `protobuf:"bytes,4,opt,name=snapshot_id,json=snapshotId,proto3,casttype=ID" json:"snapshot_id,omitempty"`
	// 'change_index' is the change index at which the snapshot ended
	ChangeIndex github_com_onosproject_onos_config_api_types_change_device.Index `protobuf:"varint,5,opt,name=change_index,json=changeIndex,proto3,casttype=github.com/onosproject/onos-config/api/types/change/device.Index" json:"change_index,omitempty"`
	// 'values' is a list of values to set
	Values []*device.PathValue `protobuf:"bytes,6,rep,name=values,proto3" json:"values,omitempty"`
}

func (m *Snapshot) Reset()         { *m = Snapshot{} }
func (m *Snapshot) String() string { return proto.CompactTextString(m) }
func (*Snapshot) ProtoMessage()    {}
func (*Snapshot) Descriptor() ([]byte, []int) {
	return fileDescriptor_0f739fc21bf8c083, []int{2}
}
func (m *Snapshot) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Snapshot) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Snapshot.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Snapshot) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Snapshot.Merge(m, src)
}
func (m *Snapshot) XXX_Size() int {
	return m.Size()
}
func (m *Snapshot) XXX_DiscardUnknown() {
	xxx_messageInfo_Snapshot.DiscardUnknown(m)
}

var xxx_messageInfo_Snapshot proto.InternalMessageInfo

func (m *Snapshot) GetID() ID {
	if m != nil {
		return m.ID
	}
	return ""
}

func (m *Snapshot) GetDeviceID() github_com_onosproject_onos_config_api_types_device.ID {
	if m != nil {
		return m.DeviceID
	}
	return ""
}

func (m *Snapshot) GetDeviceVersion() github_com_onosproject_onos_config_api_types_device.Version {
	if m != nil {
		return m.DeviceVersion
	}
	return ""
}

func (m *Snapshot) GetSnapshotID() ID {
	if m != nil {
		return m.SnapshotID
	}
	return ""
}

func (m *Snapshot) GetChangeIndex() github_com_onosproject_onos_config_api_types_change_device.Index {
	if m != nil {
		return m.ChangeIndex
	}
	return 0
}

func (m *Snapshot) GetValues() []*device.PathValue {
	if m != nil {
		return m.Values
	}
	return nil
}

func init() {
	proto.RegisterType((*DeviceSnapshot)(nil), "onos.config.snapshot.device.DeviceSnapshot")
	proto.RegisterType((*NetworkSnapshotRef)(nil), "onos.config.snapshot.device.NetworkSnapshotRef")
	proto.RegisterType((*Snapshot)(nil), "onos.config.snapshot.device.Snapshot")
}

func init() {
	proto.RegisterFile("api/types/snapshot/device/types.proto", fileDescriptor_0f739fc21bf8c083)
}

var fileDescriptor_0f739fc21bf8c083 = []byte{
	// 593 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe4, 0x54, 0x4d, 0x8b, 0xd3, 0x4e,
	0x18, 0x6f, 0xba, 0x6d, 0x37, 0x3b, 0xed, 0x7f, 0xff, 0x32, 0xac, 0x10, 0xea, 0x92, 0x94, 0xa2,
	0xd0, 0x8b, 0x13, 0xa8, 0xb8, 0xe0, 0x0b, 0xbe, 0xd4, 0xb2, 0x10, 0x10, 0x91, 0xac, 0xec, 0xb5,
	0xa6, 0xc9, 0x34, 0x1d, 0xdd, 0x66, 0x42, 0x32, 0xad, 0xf5, 0x5b, 0xec, 0x17, 0xf1, 0xe6, 0x87,
	0xd8, 0xe3, 0x82, 0x17, 0x4f, 0x51, 0xd2, 0x6f, 0xd1, 0x93, 0x64, 0x5e, 0x4a, 0x8b, 0x45, 0xe8,
	0x7a, 0xf4, 0x52, 0x9e, 0x3c, 0xf3, 0xfb, 0xfd, 0x9e, 0xf7, 0x82, 0x7b, 0x5e, 0x4c, 0x6c, 0xf6,
	0x39, 0xc6, 0xa9, 0x9d, 0x46, 0x5e, 0x9c, 0x8e, 0x29, 0xb3, 0x03, 0x3c, 0x23, 0x3e, 0x16, 0x5e,
	0x14, 0x27, 0x94, 0x51, 0x78, 0x87, 0x46, 0x34, 0x45, 0x3e, 0x8d, 0x46, 0x24, 0x44, 0x0a, 0x88,
	0x04, 0xb0, 0x69, 0x85, 0x94, 0x86, 0x17, 0xd8, 0xe6, 0xd0, 0xe1, 0x74, 0x64, 0x33, 0x32, 0xc1,
	0x29, 0xf3, 0x26, 0xb1, 0x60, 0x37, 0x8f, 0x42, 0x1a, 0x52, 0x6e, 0xda, 0x85, 0x25, 0xbd, 0x2f,
	0x43, 0xc2, 0xc6, 0xd3, 0x21, 0xf2, 0xe9, 0xc4, 0x2e, 0xe4, 0xe3, 0x84, 0x7e, 0xc0, 0x3e, 0xe3,
	0xf6, 0x7d, 0x11, 0xca, 0xde, 0x92, 0xdd, 0x5a, 0x5a, 0xcd, 0xd3, 0x9d, 0x24, 0xfc, 0xb1, 0x17,
	0x85, 0x78, 0x4b, 0x79, 0xed, 0xaf, 0x55, 0x70, 0xd8, 0xe7, 0xee, 0x33, 0x19, 0x06, 0x1e, 0x83,
	0x32, 0x09, 0x0c, 0xad, 0xa5, 0x75, 0x0e, 0x7a, 0x8d, 0x3c, 0xb3, 0xca, 0x4e, 0x7f, 0xc9, 0x7f,
	0xdd, 0x32, 0x09, 0xa0, 0x0f, 0x0e, 0x84, 0xcc, 0x80, 0x04, 0x46, 0x99, 0x83, 0x4e, 0xf3, 0xcc,
	0xd2, 0x85, 0x08, 0x87, 0x9e, 0xec, 0x94, 0x9b, 0x50, 0x43, 0x4e, 0xdf, 0xd5, 0x85, 0xe9, 0x04,
	0x70, 0x04, 0x0e, 0x65, 0x90, 0x19, 0x4e, 0x52, 0x42, 0x23, 0x63, 0x8f, 0x47, 0x7a, 0xbe, 0xcc,
	0xac, 0x27, 0x37, 0x51, 0x3f, 0x17, 0x32, 0xee, 0x7f, 0xe2, 0x5b, 0x7e, 0xc2, 0x0e, 0xd0, 0x13,
	0x3c, 0x23, 0x3c, 0x42, 0xa5, 0xa5, 0x75, 0x2a, 0xbd, 0xc6, 0x32, 0xb3, 0x74, 0x57, 0xfa, 0xdc,
	0xd5, 0x2b, 0x7c, 0x0f, 0x6e, 0x45, 0x98, 0x7d, 0xa2, 0xc9, 0xc7, 0x81, 0x9a, 0x87, 0x51, 0x6d,
	0x69, 0x9d, 0x7a, 0xd7, 0x46, 0x7f, 0xd8, 0x10, 0xf4, 0x46, 0x90, 0x54, 0x73, 0x5d, 0x3c, 0xea,
	0x55, 0xae, 0x32, 0xab, 0xe4, 0xfe, 0x1f, 0x6d, 0xbe, 0x40, 0x0a, 0x8c, 0x89, 0x37, 0x1f, 0xa8,
	0x28, 0x62, 0x64, 0x03, 0x12, 0x05, 0x78, 0x6e, 0xd4, 0x78, 0x6e, 0x27, 0xcb, 0xcc, 0xea, 0xee,
	0x52, 0x3d, 0x72, 0x0a, 0xb6, 0x7b, 0x7b, 0xe2, 0xcd, 0x65, 0x1e, 0xaf, 0xb8, 0x2a, 0x77, 0xc3,
	0xc7, 0xa0, 0x96, 0x32, 0x8f, 0x4d, 0x53, 0x63, 0x9f, 0x17, 0x72, 0xbc, 0xbd, 0x90, 0x33, 0x8e,
	0x91, 0x59, 0x4b, 0x06, 0x7c, 0x06, 0xf6, 0xfd, 0x04, 0x7b, 0x0c, 0x07, 0x86, 0xce, 0xc9, 0x4d,
	0x24, 0x4e, 0x01, 0xa9, 0x53, 0x40, 0xef, 0xd4, 0x29, 0xf4, 0xf4, 0x82, 0x7a, 0xf9, 0xc3, 0xd2,
	0x5c, 0x45, 0x2a, 0xf8, 0xd3, 0x38, 0xe0, 0xfc, 0x83, 0x5d, 0xf8, 0x92, 0xd4, 0xfe, 0xa2, 0x01,
	0xf8, 0x7b, 0x6b, 0xa1, 0xb3, 0xb6, 0xba, 0x8f, 0x56, 0xab, 0x6b, 0xef, 0xd6, 0x33, 0xb1, 0xe7,
	0xaf, 0x41, 0x55, 0xf4, 0xbe, 0xfc, 0x57, 0xbd, 0x17, 0x22, 0xed, 0x6f, 0x7b, 0x40, 0xff, 0x17,
	0x0f, 0xec, 0x21, 0xa8, 0xab, 0x45, 0x2a, 0xca, 0xa9, 0xf0, 0x20, 0x47, 0x79, 0x66, 0x01, 0xd5,
	0x8d, 0x55, 0xed, 0x40, 0x01, 0x9d, 0x00, 0x86, 0xa0, 0xb1, 0xb1, 0xff, 0x55, 0x3e, 0x83, 0x02,
	0xf9, 0xe2, 0xe6, 0xff, 0x7b, 0x72, 0x22, 0x75, 0x7f, 0xed, 0x06, 0x9e, 0x82, 0xda, 0xcc, 0xbb,
	0x98, 0xe2, 0xd4, 0xa8, 0xb5, 0xf6, 0x3a, 0xf5, 0xee, 0xdd, 0x8d, 0x1b, 0x10, 0x48, 0x75, 0xca,
	0x6f, 0x3d, 0x36, 0x3e, 0x2f, 0xc0, 0xae, 0xe4, 0xf4, 0x8c, 0xab, 0xdc, 0xd4, 0xae, 0x73, 0x53,
	0xfb, 0x99, 0x9b, 0xda, 0xe5, 0xc2, 0x2c, 0x5d, 0x2f, 0xcc, 0xd2, 0xf7, 0x85, 0x59, 0x1a, 0xd6,
	0xf8, 0x1a, 0x3f, 0xf8, 0x15, 0x00, 0x00, 0xff, 0xff, 0x4a, 0x80, 0xec, 0x9b, 0x65, 0x06, 0x00,
	0x00,
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
	n1, err1 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Updated, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Updated):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintTypes(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x4a
	n2, err2 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Created, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Created):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintTypes(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x42
	{
		size, err := m.Status.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x3a
	if m.MaxNetworkChangeIndex != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.MaxNetworkChangeIndex))
		i--
		dAtA[i] = 0x30
	}
	{
		size, err := m.NetworkSnapshot.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x2a
	if m.Revision != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.Revision))
		i--
		dAtA[i] = 0x20
	}
	if len(m.DeviceVersion) > 0 {
		i -= len(m.DeviceVersion)
		copy(dAtA[i:], m.DeviceVersion)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.DeviceVersion)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.DeviceID) > 0 {
		i -= len(m.DeviceID)
		copy(dAtA[i:], m.DeviceID)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.DeviceID)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.ID)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *NetworkSnapshotRef) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *NetworkSnapshotRef) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *NetworkSnapshotRef) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Index != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.Index))
		i--
		dAtA[i] = 0x10
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.ID)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *Snapshot) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Snapshot) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Snapshot) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Values) > 0 {
		for iNdEx := len(m.Values) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Values[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintTypes(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x32
		}
	}
	if m.ChangeIndex != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.ChangeIndex))
		i--
		dAtA[i] = 0x28
	}
	if len(m.SnapshotID) > 0 {
		i -= len(m.SnapshotID)
		copy(dAtA[i:], m.SnapshotID)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.SnapshotID)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.DeviceVersion) > 0 {
		i -= len(m.DeviceVersion)
		copy(dAtA[i:], m.DeviceVersion)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.DeviceVersion)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.DeviceID) > 0 {
		i -= len(m.DeviceID)
		copy(dAtA[i:], m.DeviceID)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.DeviceID)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.ID)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTypes(dAtA []byte, offset int, v uint64) int {
	offset -= sovTypes(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *DeviceSnapshot) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	l = len(m.DeviceID)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	l = len(m.DeviceVersion)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	if m.Revision != 0 {
		n += 1 + sovTypes(uint64(m.Revision))
	}
	l = m.NetworkSnapshot.Size()
	n += 1 + l + sovTypes(uint64(l))
	if m.MaxNetworkChangeIndex != 0 {
		n += 1 + sovTypes(uint64(m.MaxNetworkChangeIndex))
	}
	l = m.Status.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Created)
	n += 1 + l + sovTypes(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Updated)
	n += 1 + l + sovTypes(uint64(l))
	return n
}

func (m *NetworkSnapshotRef) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	if m.Index != 0 {
		n += 1 + sovTypes(uint64(m.Index))
	}
	return n
}

func (m *Snapshot) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	l = len(m.DeviceID)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	l = len(m.DeviceVersion)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	l = len(m.SnapshotID)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	if m.ChangeIndex != 0 {
		n += 1 + sovTypes(uint64(m.ChangeIndex))
	}
	if len(m.Values) > 0 {
		for _, e := range m.Values {
			l = e.Size()
			n += 1 + l + sovTypes(uint64(l))
		}
	}
	return n
}

func sovTypes(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTypes(x uint64) (n int) {
	return sovTypes(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *DeviceSnapshot) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypes
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
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
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
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DeviceID = github_com_onosproject_onos_config_api_types_device.ID(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DeviceVersion", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DeviceVersion = github_com_onosproject_onos_config_api_types_device.Version(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Revision", wireType)
			}
			m.Revision = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Revision |= Revision(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NetworkSnapshot", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.NetworkSnapshot.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxNetworkChangeIndex", wireType)
			}
			m.MaxNetworkChangeIndex = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaxNetworkChangeIndex |= github_com_onosproject_onos_config_api_types.Index(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Status.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Created", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_gogo_protobuf_types.StdTimeUnmarshal(&m.Created, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Updated", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
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
			skippy, err := skipTypes(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthTypes
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthTypes
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
func (m *NetworkSnapshotRef) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypes
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
			return fmt.Errorf("proto: NetworkSnapshotRef: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: NetworkSnapshotRef: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ID = github_com_onosproject_onos_config_api_types.ID(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Index", wireType)
			}
			m.Index = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Index |= github_com_onosproject_onos_config_api_types.Index(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipTypes(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthTypes
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthTypes
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
func (m *Snapshot) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTypes
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
			return fmt.Errorf("proto: Snapshot: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Snapshot: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
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
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DeviceID = github_com_onosproject_onos_config_api_types_device.ID(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DeviceVersion", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DeviceVersion = github_com_onosproject_onos_config_api_types_device.Version(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SnapshotID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SnapshotID = ID(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChangeIndex", wireType)
			}
			m.ChangeIndex = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ChangeIndex |= github_com_onosproject_onos_config_api_types_change_device.Index(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Values", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
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
				return ErrInvalidLengthTypes
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTypes
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Values = append(m.Values, &device.PathValue{})
			if err := m.Values[len(m.Values)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTypes(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthTypes
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthTypes
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
func skipTypes(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTypes
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
					return 0, ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowTypes
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
				return 0, ErrInvalidLengthTypes
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTypes
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTypes
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTypes        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTypes          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTypes = fmt.Errorf("proto: unexpected end of group")
)
