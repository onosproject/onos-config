// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: pkg/types/network/change/types.proto

package change

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	_ "github.com/gogo/protobuf/types"
	github_com_gogo_protobuf_types "github.com/gogo/protobuf/types"
	change "github.com/onosproject/onos-config/pkg/types/device/change"
	meta "github.com/onosproject/onos-config/pkg/types/meta"
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
	Status meta.Status `protobuf:"varint,4,opt,name=status,proto3,enum=onos.config.meta.Status" json:"status,omitempty"`
	// 'reason' indicates the reason the change failed if in the FAILED state
	Reason meta.FailureReason `protobuf:"varint,5,opt,name=reason,proto3,enum=onos.config.meta.FailureReason" json:"reason,omitempty"`
	// 'message' is an optional status message
	Message string `protobuf:"bytes,6,opt,name=message,proto3" json:"message,omitempty"`
	// 'created' is the time at which the change was created
	Created time.Time `protobuf:"bytes,7,opt,name=created,proto3,stdtime" json:"created"`
	// 'updated' is the time at which the change was last updated
	Updated time.Time `protobuf:"bytes,8,opt,name=updated,proto3,stdtime" json:"updated"`
	// 'changes' is a set of changes to apply to devices
	// The list of changes should contain only a single change per device/version pair.
	Changes []*change.Change `protobuf:"bytes,9,rep,name=changes,proto3" json:"changes,omitempty"`
}

func (m *NetworkChange) Reset()         { *m = NetworkChange{} }
func (m *NetworkChange) String() string { return proto.CompactTextString(m) }
func (*NetworkChange) ProtoMessage()    {}
func (*NetworkChange) Descriptor() ([]byte, []int) {
	return fileDescriptor_f06220d99ecb53d1, []int{0}
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

func (m *NetworkChange) GetStatus() meta.Status {
	if m != nil {
		return m.Status
	}
	return meta.Status_PENDING
}

func (m *NetworkChange) GetReason() meta.FailureReason {
	if m != nil {
		return m.Reason
	}
	return meta.FailureReason_ERROR
}

func (m *NetworkChange) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
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

func (m *NetworkChange) GetChanges() []*change.Change {
	if m != nil {
		return m.Changes
	}
	return nil
}

func init() {
	proto.RegisterType((*NetworkChange)(nil), "onos.config.network.change.NetworkChange")
}

func init() {
	proto.RegisterFile("pkg/types/network/change/types.proto", fileDescriptor_f06220d99ecb53d1)
}

var fileDescriptor_f06220d99ecb53d1 = []byte{
	// 430 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0x3f, 0x6f, 0xd4, 0x30,
	0x18, 0xc6, 0xcf, 0x77, 0xbd, 0x7f, 0x6e, 0x61, 0xb0, 0x18, 0xac, 0x08, 0x25, 0x01, 0x31, 0x64,
	0xc1, 0x41, 0xc7, 0xc0, 0x80, 0x60, 0x38, 0x50, 0xa5, 0x2e, 0x0c, 0x86, 0x2f, 0xe0, 0x4b, 0xde,
	0xba, 0xa6, 0x4d, 0x1c, 0xc5, 0x4e, 0x81, 0x6f, 0xd1, 0xcf, 0xc3, 0x27, 0xe8, 0xd8, 0x91, 0xe9,
	0x40, 0xb9, 0x6f, 0x71, 0x13, 0x8a, 0x9d, 0x40, 0x11, 0x5d, 0xba, 0x9c, 0xde, 0x7b, 0xde, 0xe7,
	0x17, 0xfb, 0xc9, 0x13, 0xfc, 0xac, 0x3a, 0x97, 0xa9, 0xfd, 0x56, 0x81, 0x49, 0x4b, 0xb0, 0x5f,
	0x74, 0x7d, 0x9e, 0x66, 0x67, 0xa2, 0x94, 0xe0, 0x45, 0x56, 0xd5, 0xda, 0x6a, 0x12, 0xe8, 0x52,
	0x1b, 0x96, 0xe9, 0xf2, 0x54, 0x49, 0xd6, 0xfb, 0x98, 0xf7, 0x05, 0x91, 0xd4, 0x5a, 0x5e, 0x40,
	0xea, 0x9c, 0x9b, 0xe6, 0x34, 0xb5, 0xaa, 0x00, 0x63, 0x45, 0x51, 0x79, 0x38, 0x78, 0x24, 0xb5,
	0xd4, 0x6e, 0x4c, 0xbb, 0xa9, 0x57, 0xdf, 0x48, 0x65, 0xcf, 0x9a, 0x0d, 0xcb, 0x74, 0x91, 0x76,
	0x4f, 0xaf, 0x6a, 0xfd, 0x19, 0x32, 0xeb, 0xe6, 0xe7, 0xfe, 0xa4, 0xf4, 0xef, 0xdd, 0x0a, 0xb0,
	0xe2, 0xf6, 0x8d, 0x82, 0xe3, 0x7b, 0xe1, 0x39, 0x5c, 0xaa, 0x0c, 0xee, 0x48, 0xf6, 0xf4, 0xfb,
	0x04, 0x3f, 0xf8, 0xe0, 0x03, 0xbd, 0x73, 0x5b, 0xf2, 0x18, 0x8f, 0x55, 0x4e, 0x51, 0x8c, 0x92,
	0xe5, 0xfa, 0xa8, 0xdd, 0x46, 0xe3, 0x93, 0xf7, 0x7b, 0xf7, 0xcb, 0xc7, 0x2a, 0x27, 0x11, 0x9e,
	0xaa, 0x32, 0x87, 0xaf, 0x74, 0x1c, 0xa3, 0xe4, 0x60, 0xbd, 0xdc, 0x6f, 0xa3, 0xe9, 0x49, 0x27,
	0x70, 0xaf, 0x93, 0x04, 0x2f, 0x6a, 0xb8, 0x54, 0x46, 0xe9, 0x92, 0x4e, 0x9c, 0xe7, 0x68, 0xbf,
	0x8d, 0x16, 0xbc, 0xd7, 0xf8, 0x9f, 0x2d, 0x79, 0x81, 0x67, 0xc6, 0x0a, 0xdb, 0x18, 0x7a, 0x10,
	0xa3, 0xe4, 0xe1, 0x8a, 0xb2, 0xdb, 0x6f, 0xb9, 0x4b, 0xcc, 0x3e, 0xba, 0x3d, 0xef, 0x7d, 0xe4,
	0x15, 0x9e, 0xd5, 0x20, 0x8c, 0x2e, 0xe9, 0xd4, 0x11, 0xd1, 0xff, 0xc4, 0xb1, 0x50, 0x17, 0x4d,
	0x0d, 0xdc, 0xd9, 0x78, 0x6f, 0x27, 0x14, 0xcf, 0x0b, 0x30, 0x46, 0x48, 0xa0, 0xb3, 0x2e, 0x18,
	0x1f, 0xfe, 0x92, 0xb7, 0x78, 0x9e, 0xd5, 0x20, 0x2c, 0xe4, 0x74, 0x1e, 0xa3, 0xe4, 0x70, 0x15,
	0x30, 0xdf, 0x27, 0x1b, 0xfa, 0x64, 0x9f, 0x86, 0x3e, 0xd7, 0x8b, 0xeb, 0x6d, 0x34, 0xba, 0xfa,
	0x19, 0x21, 0x3e, 0x40, 0x1d, 0xdf, 0x54, 0xb9, 0xe3, 0x17, 0xf7, 0xe1, 0x7b, 0x88, 0xbc, 0xc6,
	0x73, 0xdf, 0x8a, 0xa1, 0xcb, 0x78, 0x92, 0x1c, 0xae, 0x9e, 0xfc, 0x93, 0xc9, 0x17, 0xd7, 0x7f,
	0x6a, 0xcc, 0x37, 0xc4, 0x07, 0x62, 0x4d, 0xaf, 0xdb, 0x10, 0xdd, 0xb4, 0x21, 0xfa, 0xd5, 0x86,
	0xe8, 0x6a, 0x17, 0x8e, 0x6e, 0x76, 0xe1, 0xe8, 0xc7, 0x2e, 0x1c, 0x6d, 0x66, 0xee, 0xf4, 0x97,
	0xbf, 0x03, 0x00, 0x00, 0xff, 0xff, 0xe1, 0x4b, 0xd3, 0xcb, 0xdf, 0x02, 0x00, 0x00,
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
			dAtA[i] = 0x4a
		}
	}
	n1, err1 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Updated, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Updated):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintTypes(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x42
	n2, err2 := github_com_gogo_protobuf_types.StdTimeMarshalTo(m.Created, dAtA[i-github_com_gogo_protobuf_types.SizeOfStdTime(m.Created):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintTypes(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x3a
	if len(m.Message) > 0 {
		i -= len(m.Message)
		copy(dAtA[i:], m.Message)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.Message)))
		i--
		dAtA[i] = 0x32
	}
	if m.Reason != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.Reason))
		i--
		dAtA[i] = 0x28
	}
	if m.Status != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x20
	}
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
	if m.Status != 0 {
		n += 1 + sovTypes(uint64(m.Status))
	}
	if m.Reason != 0 {
		n += 1 + sovTypes(uint64(m.Reason))
	}
	l = len(m.Message)
	if l > 0 {
		n += 1 + l + sovTypes(uint64(l))
	}
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
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= meta.Status(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Reason", wireType)
			}
			m.Reason = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Reason |= meta.FailureReason(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Message", wireType)
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
			m.Message = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
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
		case 8:
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
		case 9:
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
			m.Changes = append(m.Changes, &change.Change{})
			if err := m.Changes[len(m.Changes)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
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
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
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
			if iNdEx < 0 {
				return 0, ErrInvalidLengthTypes
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowTypes
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
				next, err := skipTypes(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthTypes
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
	ErrInvalidLengthTypes = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTypes   = fmt.Errorf("proto: integer overflow")
)
