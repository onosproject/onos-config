// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: pkg/types/change/types.proto

package change

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// Reason is a reason for the status message
type Reason int32

const (
	// UNKNOWN is the default reason indicating no reason was provided
	Reason_UNKNOWN Reason = 0
	// ERROR indicates an error occurred while handling the change
	Reason_ERROR Reason = 1
	// UNAVAILABLE indicates a device was unavailable
	Reason_UNAVAILABLE Reason = 2
)

var Reason_name = map[int32]string{
	0: "UNKNOWN",
	1: "ERROR",
	2: "UNAVAILABLE",
}

var Reason_value = map[string]int32{
	"UNKNOWN":     0,
	"ERROR":       1,
	"UNAVAILABLE": 2,
}

func (x Reason) String() string {
	return proto.EnumName(Reason_name, int32(x))
}

func (Reason) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_12f7eaf625e420c6, []int{0}
}

// State indicates the state of a change
type State int32

const (
	// PENDING indicates the change is pending
	State_PENDING State = 0
	// APPLYING indicates the change is being applied
	State_APPLYING State = 1
	// SUCCEEDED indicates the change has been applied successfully
	State_SUCCEEDED State = 2
	// FAILED indicates the change failed
	State_FAILED State = 3
)

var State_name = map[int32]string{
	0: "PENDING",
	1: "APPLYING",
	2: "SUCCEEDED",
	3: "FAILED",
}

var State_value = map[string]int32{
	"PENDING":   0,
	"APPLYING":  1,
	"SUCCEEDED": 2,
	"FAILED":    3,
}

func (x State) String() string {
	return proto.EnumName(State_name, int32(x))
}

func (State) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_12f7eaf625e420c6, []int{1}
}

// Status provides status information for a change
type Status struct {
	// 'state' indicates the state of the change
	State State `protobuf:"varint,1,opt,name=state,proto3,enum=onos.config.change.State" json:"state,omitempty"`
	// 'deleted' indicates whether the object has been deleted
	Deleted bool `protobuf:"varint,2,opt,name=deleted,proto3" json:"deleted,omitempty"`
	// 'reason' is a status reason
	Reason Reason `protobuf:"varint,3,opt,name=reason,proto3,enum=onos.config.change.Reason" json:"reason,omitempty"`
	// 'message' is a status message
	Message string `protobuf:"bytes,4,opt,name=message,proto3" json:"message,omitempty"`
}

func (m *Status) Reset()         { *m = Status{} }
func (m *Status) String() string { return proto.CompactTextString(m) }
func (*Status) ProtoMessage()    {}
func (*Status) Descriptor() ([]byte, []int) {
	return fileDescriptor_12f7eaf625e420c6, []int{0}
}
func (m *Status) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Status) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Status.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Status) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Status.Merge(m, src)
}
func (m *Status) XXX_Size() int {
	return m.Size()
}
func (m *Status) XXX_DiscardUnknown() {
	xxx_messageInfo_Status.DiscardUnknown(m)
}

var xxx_messageInfo_Status proto.InternalMessageInfo

func (m *Status) GetState() State {
	if m != nil {
		return m.State
	}
	return State_PENDING
}

func (m *Status) GetDeleted() bool {
	if m != nil {
		return m.Deleted
	}
	return false
}

func (m *Status) GetReason() Reason {
	if m != nil {
		return m.Reason
	}
	return Reason_UNKNOWN
}

func (m *Status) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func init() {
	proto.RegisterEnum("onos.config.change.Reason", Reason_name, Reason_value)
	proto.RegisterEnum("onos.config.change.State", State_name, State_value)
	proto.RegisterType((*Status)(nil), "onos.config.change.Status")
}

func init() { proto.RegisterFile("pkg/types/change/types.proto", fileDescriptor_12f7eaf625e420c6) }

var fileDescriptor_12f7eaf625e420c6 = []byte{
	// 296 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x90, 0xcd, 0x4e, 0xc2, 0x40,
	0x10, 0x80, 0xbb, 0x20, 0x05, 0x06, 0x7f, 0x36, 0x7b, 0x5a, 0x8d, 0xd9, 0x10, 0x4f, 0x84, 0x43,
	0x89, 0x78, 0xf6, 0x50, 0xe8, 0x6a, 0x88, 0xcd, 0x42, 0x96, 0xa0, 0xf1, 0x58, 0x61, 0xad, 0x46,
	0xed, 0x36, 0xec, 0x7a, 0xf0, 0x2d, 0x7c, 0x05, 0xdf, 0xc6, 0x23, 0x47, 0x8f, 0xa6, 0x7d, 0x11,
	0xd3, 0x56, 0x4e, 0x7a, 0xfc, 0x32, 0xdf, 0x7c, 0x99, 0x0c, 0x1c, 0xa7, 0x4f, 0xf1, 0xc0, 0xbe,
	0xa5, 0xca, 0x0c, 0x96, 0x0f, 0x51, 0x12, 0xab, 0x0a, 0xbc, 0x74, 0xad, 0xad, 0x26, 0x44, 0x27,
	0xda, 0x78, 0x4b, 0x9d, 0xdc, 0x3f, 0xc6, 0x5e, 0x35, 0x3f, 0xf9, 0x40, 0xe0, 0xce, 0x6d, 0x64,
	0x5f, 0x0d, 0x19, 0x40, 0xc3, 0xd8, 0xc8, 0x2a, 0x8a, 0xba, 0xa8, 0xb7, 0x3f, 0x3c, 0xf4, 0xfe,
	0xea, 0x5e, 0xa1, 0x2a, 0x59, 0x79, 0x84, 0x42, 0x73, 0xa5, 0x9e, 0x95, 0x55, 0x2b, 0x5a, 0xeb,
	0xa2, 0x5e, 0x4b, 0x6e, 0x91, 0x0c, 0xc1, 0x5d, 0xab, 0xc8, 0xe8, 0x84, 0xd6, 0xcb, 0xd6, 0xd1,
	0x7f, 0x2d, 0x59, 0x1a, 0xf2, 0xd7, 0x2c, 0x6a, 0x2f, 0xca, 0x98, 0x28, 0x56, 0x74, 0xa7, 0x8b,
	0x7a, 0x6d, 0xb9, 0xc5, 0xfe, 0x29, 0xb8, 0x95, 0x4b, 0x3a, 0xd0, 0x5c, 0x88, 0x2b, 0x31, 0xbd,
	0x11, 0xd8, 0x21, 0x6d, 0x68, 0x70, 0x29, 0xa7, 0x12, 0x23, 0x72, 0x00, 0x9d, 0x85, 0xf0, 0xaf,
	0xfd, 0x49, 0xe8, 0x8f, 0x42, 0x8e, 0x6b, 0xfd, 0x73, 0x68, 0x94, 0xa7, 0x16, 0x1b, 0x33, 0x2e,
	0x82, 0x89, 0xb8, 0xc4, 0x0e, 0xd9, 0x85, 0x96, 0x3f, 0x9b, 0x85, 0xb7, 0x05, 0x21, 0xb2, 0x07,
	0xed, 0xf9, 0x62, 0x3c, 0xe6, 0x3c, 0xe0, 0x01, 0xae, 0x11, 0x00, 0xf7, 0xc2, 0x9f, 0x84, 0x3c,
	0xc0, 0xf5, 0x11, 0xfd, 0xcc, 0x18, 0xda, 0x64, 0x0c, 0x7d, 0x67, 0x0c, 0xbd, 0xe7, 0xcc, 0xd9,
	0xe4, 0xcc, 0xf9, 0xca, 0x99, 0x73, 0xe7, 0x96, 0xaf, 0x3c, 0xfb, 0x09, 0x00, 0x00, 0xff, 0xff,
	0x37, 0x07, 0xe6, 0x9a, 0x6a, 0x01, 0x00, 0x00,
}

func (m *Status) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Status) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Status) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Message) > 0 {
		i -= len(m.Message)
		copy(dAtA[i:], m.Message)
		i = encodeVarintTypes(dAtA, i, uint64(len(m.Message)))
		i--
		dAtA[i] = 0x22
	}
	if m.Reason != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.Reason))
		i--
		dAtA[i] = 0x18
	}
	if m.Deleted {
		i--
		if m.Deleted {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x10
	}
	if m.State != 0 {
		i = encodeVarintTypes(dAtA, i, uint64(m.State))
		i--
		dAtA[i] = 0x8
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
func (m *Status) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.State != 0 {
		n += 1 + sovTypes(uint64(m.State))
	}
	if m.Deleted {
		n += 2
	}
	if m.Reason != 0 {
		n += 1 + sovTypes(uint64(m.Reason))
	}
	l = len(m.Message)
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
func (m *Status) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: Status: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Status: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field State", wireType)
			}
			m.State = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTypes
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.State |= State(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
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
		case 3:
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
				m.Reason |= Reason(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
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
