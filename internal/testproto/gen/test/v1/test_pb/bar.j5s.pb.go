// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: test/v1/bar.j5s.proto

package test_pb

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	_ "github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	_ "github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	psm_j5pb "github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	date_j5t "github.com/pentops/j5/j5types/date_j5t"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type BarStatus int32

const (
	BarStatus_BAR_STATUS_UNSPECIFIED BarStatus = 0
	BarStatus_BAR_STATUS_ACTIVE      BarStatus = 1
	BarStatus_BAR_STATUS_DELETED     BarStatus = 2
)

// Enum value maps for BarStatus.
var (
	BarStatus_name = map[int32]string{
		0: "BAR_STATUS_UNSPECIFIED",
		1: "BAR_STATUS_ACTIVE",
		2: "BAR_STATUS_DELETED",
	}
	BarStatus_value = map[string]int32{
		"BAR_STATUS_UNSPECIFIED": 0,
		"BAR_STATUS_ACTIVE":      1,
		"BAR_STATUS_DELETED":     2,
	}
)

func (x BarStatus) Enum() *BarStatus {
	p := new(BarStatus)
	*p = x
	return p
}

func (x BarStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (BarStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_test_v1_bar_j5s_proto_enumTypes[0].Descriptor()
}

func (BarStatus) Type() protoreflect.EnumType {
	return &file_test_v1_bar_j5s_proto_enumTypes[0]
}

func (x BarStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use BarStatus.Descriptor instead.
func (BarStatus) EnumDescriptor() ([]byte, []int) {
	return file_test_v1_bar_j5s_proto_rawDescGZIP(), []int{0}
}

type BarKeys struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BarId      string         `protobuf:"bytes,1,opt,name=bar_id,json=barId,proto3" json:"bar_id,omitempty"`
	BarOtherId string         `protobuf:"bytes,2,opt,name=bar_other_id,json=barOtherId,proto3" json:"bar_other_id,omitempty"`
	DateKey    *date_j5t.Date `protobuf:"bytes,3,opt,name=date_key,json=dateKey,proto3" json:"date_key,omitempty"`
}

func (x *BarKeys) Reset() {
	*x = BarKeys{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_bar_j5s_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BarKeys) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BarKeys) ProtoMessage() {}

func (x *BarKeys) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_bar_j5s_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BarKeys.ProtoReflect.Descriptor instead.
func (*BarKeys) Descriptor() ([]byte, []int) {
	return file_test_v1_bar_j5s_proto_rawDescGZIP(), []int{0}
}

func (x *BarKeys) GetBarId() string {
	if x != nil {
		return x.BarId
	}
	return ""
}

func (x *BarKeys) GetBarOtherId() string {
	if x != nil {
		return x.BarOtherId
	}
	return ""
}

func (x *BarKeys) GetDateKey() *date_j5t.Date {
	if x != nil {
		return x.DateKey
	}
	return nil
}

type BarData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TenantId *string `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3,oneof" json:"tenant_id,omitempty"`
	Name     string  `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Field    string  `protobuf:"bytes,3,opt,name=field,proto3" json:"field,omitempty"`
}

func (x *BarData) Reset() {
	*x = BarData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_bar_j5s_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BarData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BarData) ProtoMessage() {}

func (x *BarData) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_bar_j5s_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BarData.ProtoReflect.Descriptor instead.
func (*BarData) Descriptor() ([]byte, []int) {
	return file_test_v1_bar_j5s_proto_rawDescGZIP(), []int{1}
}

func (x *BarData) GetTenantId() string {
	if x != nil && x.TenantId != nil {
		return *x.TenantId
	}
	return ""
}

func (x *BarData) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *BarData) GetField() string {
	if x != nil {
		return x.Field
	}
	return ""
}

type BarState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metadata *psm_j5pb.StateMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Keys     *BarKeys                `protobuf:"bytes,2,opt,name=keys,proto3" json:"keys,omitempty"`
	Data     *BarData                `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
	Status   BarStatus               `protobuf:"varint,4,opt,name=status,proto3,enum=test.v1.BarStatus" json:"status,omitempty"`
}

func (x *BarState) Reset() {
	*x = BarState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_bar_j5s_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BarState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BarState) ProtoMessage() {}

func (x *BarState) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_bar_j5s_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BarState.ProtoReflect.Descriptor instead.
func (*BarState) Descriptor() ([]byte, []int) {
	return file_test_v1_bar_j5s_proto_rawDescGZIP(), []int{2}
}

func (x *BarState) GetMetadata() *psm_j5pb.StateMetadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *BarState) GetKeys() *BarKeys {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *BarState) GetData() *BarData {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *BarState) GetStatus() BarStatus {
	if x != nil {
		return x.Status
	}
	return BarStatus_BAR_STATUS_UNSPECIFIED
}

type BarEventType struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Type:
	//
	//	*BarEventType_Created_
	//	*BarEventType_Updated_
	//	*BarEventType_Deleted_
	Type isBarEventType_Type `protobuf_oneof:"type"`
}

func (x *BarEventType) Reset() {
	*x = BarEventType{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_bar_j5s_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BarEventType) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BarEventType) ProtoMessage() {}

func (x *BarEventType) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_bar_j5s_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BarEventType.ProtoReflect.Descriptor instead.
func (*BarEventType) Descriptor() ([]byte, []int) {
	return file_test_v1_bar_j5s_proto_rawDescGZIP(), []int{3}
}

func (m *BarEventType) GetType() isBarEventType_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *BarEventType) GetCreated() *BarEventType_Created {
	if x, ok := x.GetType().(*BarEventType_Created_); ok {
		return x.Created
	}
	return nil
}

func (x *BarEventType) GetUpdated() *BarEventType_Updated {
	if x, ok := x.GetType().(*BarEventType_Updated_); ok {
		return x.Updated
	}
	return nil
}

func (x *BarEventType) GetDeleted() *BarEventType_Deleted {
	if x, ok := x.GetType().(*BarEventType_Deleted_); ok {
		return x.Deleted
	}
	return nil
}

type isBarEventType_Type interface {
	isBarEventType_Type()
}

type BarEventType_Created_ struct {
	Created *BarEventType_Created `protobuf:"bytes,1,opt,name=created,proto3,oneof"`
}

type BarEventType_Updated_ struct {
	Updated *BarEventType_Updated `protobuf:"bytes,2,opt,name=updated,proto3,oneof"`
}

type BarEventType_Deleted_ struct {
	Deleted *BarEventType_Deleted `protobuf:"bytes,3,opt,name=deleted,proto3,oneof"`
}

func (*BarEventType_Created_) isBarEventType_Type() {}

func (*BarEventType_Updated_) isBarEventType_Type() {}

func (*BarEventType_Deleted_) isBarEventType_Type() {}

type BarEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metadata *psm_j5pb.EventMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Keys     *BarKeys                `protobuf:"bytes,2,opt,name=keys,proto3" json:"keys,omitempty"`
	Event    *BarEventType           `protobuf:"bytes,3,opt,name=event,proto3" json:"event,omitempty"`
}

func (x *BarEvent) Reset() {
	*x = BarEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_bar_j5s_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BarEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BarEvent) ProtoMessage() {}

func (x *BarEvent) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_bar_j5s_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BarEvent.ProtoReflect.Descriptor instead.
func (*BarEvent) Descriptor() ([]byte, []int) {
	return file_test_v1_bar_j5s_proto_rawDescGZIP(), []int{4}
}

func (x *BarEvent) GetMetadata() *psm_j5pb.EventMetadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *BarEvent) GetKeys() *BarKeys {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *BarEvent) GetEvent() *BarEventType {
	if x != nil {
		return x.Event
	}
	return nil
}

type BarEventType_Created struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name  string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Field string `protobuf:"bytes,2,opt,name=field,proto3" json:"field,omitempty"`
}

func (x *BarEventType_Created) Reset() {
	*x = BarEventType_Created{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_bar_j5s_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BarEventType_Created) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BarEventType_Created) ProtoMessage() {}

func (x *BarEventType_Created) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_bar_j5s_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BarEventType_Created.ProtoReflect.Descriptor instead.
func (*BarEventType_Created) Descriptor() ([]byte, []int) {
	return file_test_v1_bar_j5s_proto_rawDescGZIP(), []int{3, 0}
}

func (x *BarEventType_Created) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *BarEventType_Created) GetField() string {
	if x != nil {
		return x.Field
	}
	return ""
}

type BarEventType_Updated struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name  string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Field string `protobuf:"bytes,2,opt,name=field,proto3" json:"field,omitempty"`
}

func (x *BarEventType_Updated) Reset() {
	*x = BarEventType_Updated{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_bar_j5s_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BarEventType_Updated) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BarEventType_Updated) ProtoMessage() {}

func (x *BarEventType_Updated) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_bar_j5s_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BarEventType_Updated.ProtoReflect.Descriptor instead.
func (*BarEventType_Updated) Descriptor() ([]byte, []int) {
	return file_test_v1_bar_j5s_proto_rawDescGZIP(), []int{3, 1}
}

func (x *BarEventType_Updated) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *BarEventType_Updated) GetField() string {
	if x != nil {
		return x.Field
	}
	return ""
}

type BarEventType_Deleted struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *BarEventType_Deleted) Reset() {
	*x = BarEventType_Deleted{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_bar_j5s_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BarEventType_Deleted) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BarEventType_Deleted) ProtoMessage() {}

func (x *BarEventType_Deleted) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_bar_j5s_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BarEventType_Deleted.ProtoReflect.Descriptor instead.
func (*BarEventType_Deleted) Descriptor() ([]byte, []int) {
	return file_test_v1_bar_j5s_proto_rawDescGZIP(), []int{3, 2}
}

var File_test_v1_bar_j5s_proto protoreflect.FileDescriptor

var file_test_v1_bar_j5s_proto_rawDesc = []byte{
	0x0a, 0x15, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x62, 0x61, 0x72, 0x2e, 0x6a, 0x35,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31,
	0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x6a,
	0x35, 0x2f, 0x65, 0x78, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x6a, 0x35, 0x2f, 0x6c,
	0x69, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1a, 0x6a, 0x35, 0x2f, 0x73, 0x74, 0x61,
	0x74, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x6a, 0x35, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x64,
	0x61, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xd5, 0x01, 0x0a, 0x07, 0x42, 0x61, 0x72, 0x4b, 0x65, 0x79, 0x73, 0x12, 0x33, 0x0a,
	0x06, 0x62, 0x61, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x1c, 0xba,
	0x48, 0x08, 0xc8, 0x01, 0x01, 0x72, 0x03, 0xb0, 0x01, 0x01, 0xc2, 0xff, 0x8e, 0x02, 0x05, 0xb2,
	0x02, 0x02, 0x08, 0x02, 0xea, 0x85, 0x8f, 0x02, 0x02, 0x08, 0x01, 0x52, 0x05, 0x62, 0x61, 0x72,
	0x49, 0x64, 0x12, 0x3e, 0x0a, 0x0c, 0x62, 0x61, 0x72, 0x5f, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x5f,
	0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x1c, 0xba, 0x48, 0x08, 0xc8, 0x01, 0x01,
	0x72, 0x03, 0xb0, 0x01, 0x01, 0xc2, 0xff, 0x8e, 0x02, 0x05, 0xb2, 0x02, 0x02, 0x08, 0x02, 0xea,
	0x85, 0x8f, 0x02, 0x02, 0x08, 0x01, 0x52, 0x0a, 0x62, 0x61, 0x72, 0x4f, 0x74, 0x68, 0x65, 0x72,
	0x49, 0x64, 0x12, 0x40, 0x0a, 0x08, 0x64, 0x61, 0x74, 0x65, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x6a, 0x35, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e,
	0x64, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x61, 0x74, 0x65, 0x42, 0x0d, 0xba, 0x48,
	0x03, 0xc8, 0x01, 0x01, 0xea, 0x85, 0x8f, 0x02, 0x02, 0x08, 0x01, 0x52, 0x07, 0x64, 0x61, 0x74,
	0x65, 0x4b, 0x65, 0x79, 0x3a, 0x13, 0xc2, 0xff, 0x8e, 0x02, 0x02, 0x52, 0x00, 0xea, 0x85, 0x8f,
	0x02, 0x07, 0x0a, 0x03, 0x62, 0x61, 0x72, 0x10, 0x01, 0x22, 0xd1, 0x01, 0x0a, 0x07, 0x42, 0x61,
	0x72, 0x44, 0x61, 0x74, 0x61, 0x12, 0x34, 0x0a, 0x09, 0x74, 0x65, 0x6e, 0x61, 0x6e, 0x74, 0x5f,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x12, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0,
	0x01, 0x01, 0xc2, 0xff, 0x8e, 0x02, 0x05, 0xb2, 0x02, 0x02, 0x08, 0x02, 0x48, 0x00, 0x52, 0x08,
	0x74, 0x65, 0x6e, 0x61, 0x6e, 0x74, 0x49, 0x64, 0x88, 0x01, 0x01, 0x12, 0x34, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x20, 0xc2, 0xff, 0x8e, 0x02, 0x03,
	0xf2, 0x01, 0x00, 0x8a, 0xf7, 0x98, 0xc6, 0x02, 0x12, 0x72, 0x10, 0x0a, 0x0e, 0x52, 0x0c, 0x08,
	0x01, 0x12, 0x08, 0x74, 0x73, 0x76, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x37, 0x0a, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x42, 0x21, 0xc2, 0xff, 0x8e, 0x02, 0x03, 0xf2, 0x01, 0x00, 0x8a, 0xf7, 0x98, 0xc6, 0x02, 0x13,
	0x72, 0x11, 0x0a, 0x0f, 0x52, 0x0d, 0x08, 0x01, 0x12, 0x09, 0x74, 0x73, 0x76, 0x5f, 0x66, 0x69,
	0x65, 0x6c, 0x64, 0x52, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x3a, 0x13, 0xc2, 0xff, 0x8e, 0x02,
	0x02, 0x52, 0x00, 0xea, 0x85, 0x8f, 0x02, 0x07, 0x0a, 0x03, 0x62, 0x61, 0x72, 0x10, 0x04, 0x42,
	0x0c, 0x0a, 0x0a, 0x5f, 0x74, 0x65, 0x6e, 0x61, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x22, 0x9f, 0x02,
	0x0a, 0x08, 0x42, 0x61, 0x72, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x45, 0x0a, 0x08, 0x6d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x6a,
	0x35, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65,
	0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x42, 0x0d, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01,
	0xc2, 0xff, 0x8e, 0x02, 0x02, 0x52, 0x00, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0x12, 0x35, 0x0a, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x10, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x61, 0x72, 0x4b, 0x65, 0x79,
	0x73, 0x42, 0x0f, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x52, 0x02,
	0x08, 0x01, 0x52, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x12, 0x33, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31,
	0x2e, 0x42, 0x61, 0x72, 0x44, 0x61, 0x74, 0x61, 0x42, 0x0d, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01,
	0xc2, 0xff, 0x8e, 0x02, 0x02, 0x52, 0x00, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x4b, 0x0a,
	0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x12, 0x2e,
	0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x61, 0x72, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x42, 0x1f, 0xba, 0x48, 0x08, 0xc8, 0x01, 0x01, 0x82, 0x01, 0x02, 0x10, 0x01, 0xc2, 0xff,
	0x8e, 0x02, 0x02, 0x5a, 0x00, 0x8a, 0xf7, 0x98, 0xc6, 0x02, 0x07, 0xa2, 0x01, 0x04, 0x52, 0x02,
	0x08, 0x01, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x3a, 0x13, 0xc2, 0xff, 0x8e, 0x02,
	0x02, 0x52, 0x00, 0xea, 0x85, 0x8f, 0x02, 0x07, 0x0a, 0x03, 0x62, 0x61, 0x72, 0x10, 0x02, 0x22,
	0xa3, 0x03, 0x0a, 0x0c, 0x42, 0x61, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65,
	0x12, 0x42, 0x0a, 0x07, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1d, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x61, 0x72, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64,
	0x42, 0x07, 0xc2, 0xff, 0x8e, 0x02, 0x02, 0x52, 0x00, 0x48, 0x00, 0x52, 0x07, 0x63, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x64, 0x12, 0x42, 0x0a, 0x07, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e,
	0x42, 0x61, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x2e, 0x55, 0x70, 0x64,
	0x61, 0x74, 0x65, 0x64, 0x42, 0x07, 0xc2, 0xff, 0x8e, 0x02, 0x02, 0x52, 0x00, 0x48, 0x00, 0x52,
	0x07, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x12, 0x42, 0x0a, 0x07, 0x64, 0x65, 0x6c, 0x65,
	0x74, 0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x74, 0x65, 0x73, 0x74,
	0x2e, 0x76, 0x31, 0x2e, 0x42, 0x61, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65,
	0x2e, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x42, 0x07, 0xc2, 0xff, 0x8e, 0x02, 0x02, 0x52,
	0x00, 0x48, 0x00, 0x52, 0x07, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x1a, 0x50, 0x0a, 0x07,
	0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x12, 0x1c, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xc2, 0xff, 0x8e, 0x02, 0x03, 0xf2, 0x01, 0x00, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1e, 0x0a, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xc2, 0xff, 0x8e, 0x02, 0x03, 0xf2, 0x01, 0x00, 0x52, 0x05,
	0x66, 0x69, 0x65, 0x6c, 0x64, 0x3a, 0x07, 0xc2, 0xff, 0x8e, 0x02, 0x02, 0x52, 0x00, 0x1a, 0x50,
	0x0a, 0x07, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x64, 0x12, 0x1c, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xc2, 0xff, 0x8e, 0x02, 0x03, 0xf2, 0x01,
	0x00, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1e, 0x0a, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xc2, 0xff, 0x8e, 0x02, 0x03, 0xf2, 0x01, 0x00,
	0x52, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x3a, 0x07, 0xc2, 0xff, 0x8e, 0x02, 0x02, 0x52, 0x00,
	0x1a, 0x12, 0x0a, 0x07, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x3a, 0x07, 0xc2, 0xff, 0x8e,
	0x02, 0x02, 0x52, 0x00, 0x3a, 0x07, 0xc2, 0xff, 0x8e, 0x02, 0x02, 0x5a, 0x00, 0x42, 0x06, 0x0a,
	0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0xe6, 0x01, 0x0a, 0x08, 0x42, 0x61, 0x72, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x12, 0x45, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x6a, 0x35, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e,
	0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x42, 0x0d, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0xc2, 0xff, 0x8e, 0x02, 0x02, 0x52, 0x00, 0x52,
	0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x35, 0x0a, 0x04, 0x6b, 0x65, 0x79,
	0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76,
	0x31, 0x2e, 0x42, 0x61, 0x72, 0x4b, 0x65, 0x79, 0x73, 0x42, 0x0f, 0xba, 0x48, 0x03, 0xc8, 0x01,
	0x01, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x52, 0x02, 0x08, 0x01, 0x52, 0x04, 0x6b, 0x65, 0x79, 0x73,
	0x12, 0x47, 0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x15, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x61, 0x72, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x42, 0x1a, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0xc2, 0xff,
	0x8e, 0x02, 0x02, 0x62, 0x00, 0x8a, 0xf7, 0x98, 0xc6, 0x02, 0x07, 0xaa, 0x01, 0x04, 0x52, 0x02,
	0x08, 0x01, 0x52, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x3a, 0x13, 0xc2, 0xff, 0x8e, 0x02, 0x02,
	0x52, 0x00, 0xea, 0x85, 0x8f, 0x02, 0x07, 0x0a, 0x03, 0x62, 0x61, 0x72, 0x10, 0x03, 0x2a, 0x56,
	0x0a, 0x09, 0x42, 0x61, 0x72, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1a, 0x0a, 0x16, 0x42,
	0x41, 0x52, 0x5f, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43,
	0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x15, 0x0a, 0x11, 0x42, 0x41, 0x52, 0x5f, 0x53,
	0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x56, 0x45, 0x10, 0x01, 0x12, 0x16,
	0x0a, 0x12, 0x42, 0x41, 0x52, 0x5f, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x44, 0x45, 0x4c,
	0x45, 0x54, 0x45, 0x44, 0x10, 0x02, 0x42, 0x46, 0x5a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c,
	0x2f, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x74,
	0x65, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x70, 0x62, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_test_v1_bar_j5s_proto_rawDescOnce sync.Once
	file_test_v1_bar_j5s_proto_rawDescData = file_test_v1_bar_j5s_proto_rawDesc
)

func file_test_v1_bar_j5s_proto_rawDescGZIP() []byte {
	file_test_v1_bar_j5s_proto_rawDescOnce.Do(func() {
		file_test_v1_bar_j5s_proto_rawDescData = protoimpl.X.CompressGZIP(file_test_v1_bar_j5s_proto_rawDescData)
	})
	return file_test_v1_bar_j5s_proto_rawDescData
}

var file_test_v1_bar_j5s_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_test_v1_bar_j5s_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_test_v1_bar_j5s_proto_goTypes = []interface{}{
	(BarStatus)(0),                 // 0: test.v1.BarStatus
	(*BarKeys)(nil),                // 1: test.v1.BarKeys
	(*BarData)(nil),                // 2: test.v1.BarData
	(*BarState)(nil),               // 3: test.v1.BarState
	(*BarEventType)(nil),           // 4: test.v1.BarEventType
	(*BarEvent)(nil),               // 5: test.v1.BarEvent
	(*BarEventType_Created)(nil),   // 6: test.v1.BarEventType.Created
	(*BarEventType_Updated)(nil),   // 7: test.v1.BarEventType.Updated
	(*BarEventType_Deleted)(nil),   // 8: test.v1.BarEventType.Deleted
	(*date_j5t.Date)(nil),          // 9: j5.types.date.v1.Date
	(*psm_j5pb.StateMetadata)(nil), // 10: j5.state.v1.StateMetadata
	(*psm_j5pb.EventMetadata)(nil), // 11: j5.state.v1.EventMetadata
}
var file_test_v1_bar_j5s_proto_depIdxs = []int32{
	9,  // 0: test.v1.BarKeys.date_key:type_name -> j5.types.date.v1.Date
	10, // 1: test.v1.BarState.metadata:type_name -> j5.state.v1.StateMetadata
	1,  // 2: test.v1.BarState.keys:type_name -> test.v1.BarKeys
	2,  // 3: test.v1.BarState.data:type_name -> test.v1.BarData
	0,  // 4: test.v1.BarState.status:type_name -> test.v1.BarStatus
	6,  // 5: test.v1.BarEventType.created:type_name -> test.v1.BarEventType.Created
	7,  // 6: test.v1.BarEventType.updated:type_name -> test.v1.BarEventType.Updated
	8,  // 7: test.v1.BarEventType.deleted:type_name -> test.v1.BarEventType.Deleted
	11, // 8: test.v1.BarEvent.metadata:type_name -> j5.state.v1.EventMetadata
	1,  // 9: test.v1.BarEvent.keys:type_name -> test.v1.BarKeys
	4,  // 10: test.v1.BarEvent.event:type_name -> test.v1.BarEventType
	11, // [11:11] is the sub-list for method output_type
	11, // [11:11] is the sub-list for method input_type
	11, // [11:11] is the sub-list for extension type_name
	11, // [11:11] is the sub-list for extension extendee
	0,  // [0:11] is the sub-list for field type_name
}

func init() { file_test_v1_bar_j5s_proto_init() }
func file_test_v1_bar_j5s_proto_init() {
	if File_test_v1_bar_j5s_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_test_v1_bar_j5s_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BarKeys); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_v1_bar_j5s_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BarData); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_v1_bar_j5s_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BarState); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_v1_bar_j5s_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BarEventType); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_v1_bar_j5s_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BarEvent); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_v1_bar_j5s_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BarEventType_Created); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_v1_bar_j5s_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BarEventType_Updated); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_v1_bar_j5s_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BarEventType_Deleted); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_test_v1_bar_j5s_proto_msgTypes[1].OneofWrappers = []interface{}{}
	file_test_v1_bar_j5s_proto_msgTypes[3].OneofWrappers = []interface{}{
		(*BarEventType_Created_)(nil),
		(*BarEventType_Updated_)(nil),
		(*BarEventType_Deleted_)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_test_v1_bar_j5s_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_test_v1_bar_j5s_proto_goTypes,
		DependencyIndexes: file_test_v1_bar_j5s_proto_depIdxs,
		EnumInfos:         file_test_v1_bar_j5s_proto_enumTypes,
		MessageInfos:      file_test_v1_bar_j5s_proto_msgTypes,
	}.Build()
	File_test_v1_bar_j5s_proto = out.File
	file_test_v1_bar_j5s_proto_rawDesc = nil
	file_test_v1_bar_j5s_proto_goTypes = nil
	file_test_v1_bar_j5s_proto_depIdxs = nil
}
