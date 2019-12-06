// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: api/types/change/network/types.proto

package network

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	github_com_gogo_protobuf_types "github.com/gogo/protobuf/types"
	change "github.com/onosproject/onos-config/api/types/change"
	device "github.com/onosproject/onos-config/api/types/change/device"
	github_com_onosproject_onos_config_api_types_change_device "github.com/onosproject/onos-config/api/types/change/device"
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

// NetworkChange specifies the configuration for a network change
// A network change is a configuration change that spans multiple devices. The change contains a list of
// per-device changes to be applied to the network.
type NetworkChange struct {
	// 'id' is the unique identifier of the change
	// This field should be set prior to persisting the object.
	ID ID `protobuf:"bytes,1,opt,name=id,proto3,casttype=ID" json:"id,omitempty"`
	// 'index' is a monotonically increasing, globally unique index of the change
	// The index is provided by the store, is static and unique for each unique change identifier,
	// and should not be modified by client code.
	Index Index `protobuf:"varint,2,opt,name=index,proto3,casttype=Index" json:"index,omitempty"`
	// 'revision' is the change revision number
	// The revision number is provided by the store and should not be modified by client code.
	// Each unique state of the change will be assigned a unique revision number which can be
	// used for optimistic concurrency control when updating or deleting the change state.
	Revision Revision `protobuf:"varint,3,opt,name=revision,proto3,casttype=Revision" json:"revision,omitempty"`
	// 'status' is the current lifecycle status of the change
	Status change.Status `protobuf:"bytes,4,opt,name=status,proto3" json:"status"`
	// 'attempt' indicates the number of attempts to apply the change
	Attempt uint32 `protobuf:"varint,10,opt,name=attempt,proto3" json:"attempt,omitempty"`
	// 'created' is the time at which the change was created
	Created time.Time `protobuf:"bytes,5,opt,name=created,proto3,stdtime" json:"created"`
	// 'updated' is the time at which the change was last updated
	Updated time.Time `protobuf:"bytes,6,opt,name=updated,proto3,stdtime" json:"updated"`
	// 'changes' is a set of changes to apply to devices
	// The list of changes should contain only a single change per device/version pair.
	Changes []*device.Change `protobuf:"bytes,7,rep,name=changes,proto3" json:"changes,omitempty"`
	// 'refs' is a set of references to stored device changes
	Refs []*DeviceChangeRef `protobuf:"bytes,8,rep,name=refs,proto3" json:"refs,omitempty"`
	// 'deleted' is a flag indicating whether this change is being deleted by a snapshot
	Deleted bool `protobuf:"varint,9,opt,name=deleted,proto3" json:"deleted,omitempty"`
}

func (m *NetworkChange) Reset()         { *m = NetworkChange{} }
func (m *NetworkChange) String() string { return proto.CompactTextString(m) }
func (*NetworkChange) ProtoMessage()    {}
func (*NetworkChange) Descriptor() ([]byte, []int) {
	return fileDescriptor_6dd0d36e65f2772f, []int{0}
}
func (m *NetworkChange) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *NetworkChange) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_NetworkChange.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *NetworkChange) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NetworkChange.Merge(m, src)
}
func (m *NetworkChange) XXX_Size() int {
	return m.Size()
}
func (m *NetworkChange) XXX_DiscardUnknown() {
	xxx_messageInfo_NetworkChange.DiscardUnknown(m)
}

var xxx_messageInfo_NetworkChange proto.InternalMessageInfo

func (m *NetworkChange) GetID() ID {
	if m != nil {
		return m.ID
	}
	return ""
}

func (m *NetworkChange) GetIndex() Index {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *NetworkChange) GetRevision() Revision {
	if m != nil {
		return m.Revision
	}
	return 0
}

func (m *NetworkChange) GetStatus() change.Status {
	if m != nil {
		return m.Status
	}
	return change.Status{}
}

func (m *NetworkChange) GetAttempt() uint32 {
	if m != nil {
		return m.Attempt
	}
	return 0
}

func (m *NetworkChange) GetCreated() time.Time {
	if m != nil {
		return m.Created
	}
	return time.Time{}
}

func (m *NetworkChange) GetUpdated() time.Time {
	if m != nil {
		return m.Updated
	}
	return time.Time{}
}

func (m *NetworkChange) GetChanges() []*device.Change {
	if m != nil {
		return m.Changes
	}
	return nil
}

func (m *NetworkChange) GetRefs() []*DeviceChangeRef {
	if m != nil {
		return m.Refs
	}
	return nil
}

func (m *NetworkChange) GetDeleted() bool {
	if m != nil {
		return m.Deleted
	}
	return false
}

// DeviceChangeRef is a reference to a device change
type DeviceChangeRef struct {
	// 'device_change_id' is the unique identifier of the device change
	DeviceChangeID github_com_onosproject_onos_config_api_types_change_device.ID `protobuf:"bytes,1,opt,name=device_change_id,json=deviceChangeId,proto3,casttype=github.com/onosproject/onos-config/api/types/change/device.ID" json:"device_change_id,omitempty"`
}

func (m *DeviceChangeRef) Reset()         { *m = DeviceChangeRef{} }
func (m *DeviceChangeRef) String() string { return proto.CompactTextString(m) }
func (*DeviceChangeRef) ProtoMessage()    {}
func (*DeviceChangeRef) Descriptor() ([]byte, []int) {
	return fileDescriptor_6dd0d36e65f2772f, []int{1}
}
func (m *DeviceChangeRef) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *DeviceChangeRef) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_DeviceChangeRef.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *DeviceChangeRef) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeviceChangeRef.Merge(m, src)
}
func (m *DeviceChangeRef) XXX_Size() int {
	return m.Size()
}
func (m *DeviceChangeRef) XXX_DiscardUnknown() {
	xxx_messageInfo_DeviceChangeRef.DiscardUnknown(m)
}

var xxx_messageInfo_DeviceChangeRef proto.InternalMessageInfo

func (m *DeviceChangeRef) GetDeviceChangeID() github_com_onosproject_onos_config_api_types_change_device.ID {
	if m != nil {
		return m.DeviceChangeID
	}
	return ""
}

func init() {
	proto.RegisterType((*NetworkChange)(nil), "onos.config.change.network.NetworkChange")
	proto.RegisterType((*DeviceChangeRef)(nil), "onos.config.change.network.DeviceChangeRef")
}

func init() {
	proto.RegisterFile("api/types/change/network/types.proto", fileDescriptor_6dd0d36e65f2772f)
}

var fileDescriptor_6dd0d36e65f2772f = []byte{
	// 476 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x92, 0xcd, 0x6e, 0xd4, 0x30,
	0x10, 0xc7, 0xd7, 0xfb, 0x99, 0x75, 0x3f, 0x40, 0x16, 0x07, 0x2b, 0x42, 0x49, 0xa8, 0x38, 0x44,
	0x42, 0x38, 0x52, 0xb9, 0x20, 0x21, 0xa8, 0x14, 0x56, 0x48, 0x7b, 0xe1, 0xe0, 0x72, 0xaf, 0xb2,
	0xb1, 0x37, 0x35, 0x74, 0xe3, 0x28, 0x71, 0x0a, 0x1c, 0x78, 0x87, 0x7d, 0xac, 0x1e, 0x7b, 0xe4,
	0x14, 0x50, 0x96, 0xa7, 0xd8, 0x13, 0x8a, 0x9d, 0x54, 0x0b, 0x94, 0x43, 0x7b, 0x89, 0xc6, 0x93,
	0xff, 0xef, 0xef, 0xf1, 0xcc, 0xc0, 0xa7, 0x51, 0x26, 0x02, 0xf5, 0x35, 0xe3, 0x45, 0x10, 0x9f,
	0x47, 0x69, 0xc2, 0x83, 0x94, 0xab, 0xcf, 0x32, 0xff, 0x64, 0x92, 0x24, 0xcb, 0xa5, 0x92, 0xc8,
	0x96, 0xa9, 0x2c, 0x48, 0x2c, 0xd3, 0xa5, 0x48, 0x88, 0xd1, 0x91, 0x56, 0x67, 0xbb, 0x89, 0x94,
	0xc9, 0x05, 0x0f, 0xb4, 0x72, 0x51, 0x2e, 0x03, 0x25, 0x56, 0xbc, 0x50, 0xd1, 0x2a, 0x33, 0xb0,
	0xfd, 0x28, 0x91, 0x89, 0xd4, 0x61, 0xd0, 0x44, 0x6d, 0xf6, 0x24, 0x11, 0xea, 0xbc, 0x5c, 0x90,
	0x58, 0xae, 0x82, 0xc6, 0x3d, 0xcb, 0xe5, 0x47, 0x1e, 0x2b, 0x1d, 0x3f, 0x37, 0x37, 0x05, 0xff,
	0xd4, 0xb6, 0x53, 0x93, 0xfd, 0xee, 0x3e, 0x06, 0x8c, 0x5f, 0x8a, 0xf8, 0x0f, 0x9f, 0xa3, 0x5f,
	0x03, 0x78, 0xf0, 0xde, 0xbc, 0xe5, 0xad, 0x16, 0xa1, 0xc7, 0xb0, 0x2f, 0x18, 0x06, 0x1e, 0xf0,
	0xa7, 0xe1, 0x7e, 0x5d, 0xb9, 0xfd, 0xf9, 0x6c, 0xab, 0xbf, 0xb4, 0x2f, 0x18, 0x72, 0xe1, 0x48,
	0xa4, 0x8c, 0x7f, 0xc1, 0x7d, 0x0f, 0xf8, 0xc3, 0x70, 0xba, 0xad, 0xdc, 0xd1, 0xbc, 0x49, 0x50,
	0x93, 0x47, 0x3e, 0xb4, 0x72, 0x7e, 0x29, 0x0a, 0x21, 0x53, 0x3c, 0xd0, 0x9a, 0xfd, 0x6d, 0xe5,
	0x5a, 0xb4, 0xcd, 0xd1, 0x9b, 0xbf, 0xe8, 0x25, 0x1c, 0x17, 0x2a, 0x52, 0x65, 0x81, 0x87, 0x1e,
	0xf0, 0xf7, 0x8e, 0x6d, 0x72, 0x4b, 0x9f, 0x4f, 0xb5, 0x22, 0x1c, 0x5e, 0x55, 0x6e, 0x8f, 0xb6,
	0x7a, 0x84, 0xe1, 0x24, 0x52, 0x8a, 0xaf, 0x32, 0x85, 0xa1, 0x07, 0xfc, 0x03, 0xda, 0x1d, 0xd1,
	0x1b, 0x38, 0x89, 0x73, 0x1e, 0x29, 0xce, 0xf0, 0xa8, 0x35, 0x35, 0x03, 0x22, 0xdd, 0x80, 0xc8,
	0x87, 0x6e, 0x40, 0xa1, 0xd5, 0x98, 0xae, 0x7f, 0xb8, 0x80, 0x76, 0x50, 0xc3, 0x97, 0x19, 0xd3,
	0xfc, 0xf8, 0x2e, 0x7c, 0x0b, 0xa1, 0x57, 0x70, 0x62, 0x0a, 0x2f, 0xf0, 0xc4, 0x1b, 0xf8, 0x7b,
	0xc7, 0x4f, 0x6e, 0x7b, 0x94, 0x99, 0x03, 0x31, 0x0d, 0xa7, 0x1d, 0x81, 0x4e, 0xe0, 0x30, 0xe7,
	0xcb, 0x02, 0x5b, 0x9a, 0x7c, 0x46, 0xfe, 0xbf, 0x76, 0x64, 0xa6, 0x1d, 0x5a, 0x03, 0xbe, 0xa4,
	0x1a, 0x6c, 0xfa, 0xc2, 0xf8, 0x05, 0x6f, 0xaa, 0x9f, 0x7a, 0xc0, 0xb7, 0x68, 0x77, 0x3c, 0x5a,
	0x03, 0xf8, 0xe0, 0x2f, 0x06, 0x7d, 0x83, 0x0f, 0x4d, 0x21, 0x67, 0xc6, 0xfc, 0xec, 0x66, 0xec,
	0xa7, 0x75, 0xe5, 0x1e, 0xee, 0xca, 0xf5, 0x0a, 0xbc, 0xbe, 0xff, 0xca, 0x91, 0xf9, 0x8c, 0x1e,
	0xb2, 0x5d, 0x43, 0x16, 0xe2, 0xab, 0xda, 0x01, 0xd7, 0xb5, 0x03, 0x7e, 0xd6, 0x0e, 0x58, 0x6f,
	0x9c, 0xde, 0xf5, 0xc6, 0xe9, 0x7d, 0xdf, 0x38, 0xbd, 0xc5, 0x58, 0xf7, 0xfa, 0xc5, 0xef, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xf6, 0xd9, 0xac, 0x7a, 0x9e, 0x03, 0x00, 0x00,
}

func (m *NetworkChange) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *NetworkChange) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *NetworkChange) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Attempt != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.Attempt))
		i--
		dAtA[i] = 0x50
	}
	if m.Deleted {
		i--
		if m.Deleted {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x48
	}
	if len(m.Refs) > 0 {
		for iNdEx := len(m.Refs) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Refs[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintTypes(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x42
		}
	}
	if len(m.Changes) > 0 {
		for iNdEx := len(m.Changes) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Changes[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintTypes(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x3a
		}
	}
	n1, err1 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Updated, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Updated):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintTypes(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x32
	n2, err2 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Created, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Created):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintTypes(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x2a
	{
		size, err := m.Status.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTypes(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x22
	if m.Revision != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.Revision))
		i--
		dAtA[i] = 0x18
	}
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

func (m *DeviceChangeRef) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *DeviceChangeRef) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *DeviceChangeRef) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.DeviceChangeID) > 0 {
		i -= len(m.DeviceChangeID)
		copy(dAtA[i:], m.DeviceChangeID)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.DeviceChangeID)))
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
func (m *NetworkChange) Size() (n int) {
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
	if m.Revision != 0 {
		n += 1 + sovTypes(uint64(m.Revision))
	}
	l = m.Status.Size()
	n += 1 + l + sovTypes(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Created)
	n += 1 + l + sovTypes(uint64(l))
	l = github_com_gogo_protobuf_types.SizeOfStdTime(m.Updated)
	n += 1 + l + sovTypes(uint64(l))
	if len(m.Changes) > 0 {
		for _, e := range m.Changes {
			l = e.Size()
			n += 1 + l + sovTypes(uint64(l))
		}
	}
	if len(m.Refs) > 0 {
		for _, e := range m.Refs {
			l = e.Size()
			n += 1 + l + sovTypes(uint64(l))
		}
	}
	if m.Deleted {
		n += 2
	}
	if m.Attempt != 0 {
		n += 1 + sovTypes(uint64(m.Attempt))
	}
	return n
}

func (m *DeviceChangeRef) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.DeviceChangeID)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
	return n
}

func sovTypes(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTypes(x uint64) (n int) {
	return sovTypes(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *NetworkChange) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: NetworkChange: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: NetworkChange: illegal tag %d (wire type %d)", fieldNum, wire)
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
				m.Index |= Index(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
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
		case 4:
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
		case 5:
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
		case 6:
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
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Changes", wireType)
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
			m.Changes = append(m.Changes, &device.Change{})
			if err := m.Changes[len(m.Changes)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Refs", wireType)
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
			m.Refs = append(m.Refs, &DeviceChangeRef{})
			if err := m.Refs[len(m.Refs)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Deleted", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Deleted = bool(v != 0)
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Attempt", wireType)
			}
			m.Attempt = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Attempt |= uint32(b&0x7F) << shift
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
func (m *DeviceChangeRef) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: DeviceChangeRef: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: DeviceChangeRef: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DeviceChangeID", wireType)
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
			m.DeviceChangeID = github_com_onosproject_onos_config_api_types_change_device.ID(dAtA[iNdEx:postIndex])
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
