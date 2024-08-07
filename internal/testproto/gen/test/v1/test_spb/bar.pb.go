// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: test/v1/service/bar.proto

package test_spb

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	_ "github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	list_j5pb "github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	test_pb "github.com/pentops/protostate/internal/testproto/gen/test/v1/test_pb"
	_ "google.golang.org/genproto/googleapis/api/annotations"
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

type GetBarRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BarId string `protobuf:"bytes,1,opt,name=bar_id,json=barId,proto3" json:"bar_id,omitempty"`
}

func (x *GetBarRequest) Reset() {
	*x = GetBarRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_service_bar_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetBarRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBarRequest) ProtoMessage() {}

func (x *GetBarRequest) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_service_bar_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBarRequest.ProtoReflect.Descriptor instead.
func (*GetBarRequest) Descriptor() ([]byte, []int) {
	return file_test_v1_service_bar_proto_rawDescGZIP(), []int{0}
}

func (x *GetBarRequest) GetBarId() string {
	if x != nil {
		return x.BarId
	}
	return ""
}

type GetBarResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	State  *test_pb.BarState   `protobuf:"bytes,1,opt,name=state,proto3" json:"state,omitempty"`
	Events []*test_pb.BarEvent `protobuf:"bytes,2,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *GetBarResponse) Reset() {
	*x = GetBarResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_service_bar_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetBarResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBarResponse) ProtoMessage() {}

func (x *GetBarResponse) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_service_bar_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBarResponse.ProtoReflect.Descriptor instead.
func (*GetBarResponse) Descriptor() ([]byte, []int) {
	return file_test_v1_service_bar_proto_rawDescGZIP(), []int{1}
}

func (x *GetBarResponse) GetState() *test_pb.BarState {
	if x != nil {
		return x.State
	}
	return nil
}

func (x *GetBarResponse) GetEvents() []*test_pb.BarEvent {
	if x != nil {
		return x.Events
	}
	return nil
}

type ListBarsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TenantId *string                 `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3,oneof" json:"tenant_id,omitempty"`
	Page     *list_j5pb.PageRequest  `protobuf:"bytes,2,opt,name=page,proto3" json:"page,omitempty"`
	Query    *list_j5pb.QueryRequest `protobuf:"bytes,3,opt,name=query,proto3" json:"query,omitempty"`
}

func (x *ListBarsRequest) Reset() {
	*x = ListBarsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_service_bar_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListBarsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBarsRequest) ProtoMessage() {}

func (x *ListBarsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_service_bar_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBarsRequest.ProtoReflect.Descriptor instead.
func (*ListBarsRequest) Descriptor() ([]byte, []int) {
	return file_test_v1_service_bar_proto_rawDescGZIP(), []int{2}
}

func (x *ListBarsRequest) GetTenantId() string {
	if x != nil && x.TenantId != nil {
		return *x.TenantId
	}
	return ""
}

func (x *ListBarsRequest) GetPage() *list_j5pb.PageRequest {
	if x != nil {
		return x.Page
	}
	return nil
}

func (x *ListBarsRequest) GetQuery() *list_j5pb.QueryRequest {
	if x != nil {
		return x.Query
	}
	return nil
}

type ListBarsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Bars []*test_pb.BarState     `protobuf:"bytes,1,rep,name=bars,proto3" json:"bars,omitempty"`
	Page *list_j5pb.PageResponse `protobuf:"bytes,2,opt,name=page,proto3" json:"page,omitempty"`
}

func (x *ListBarsResponse) Reset() {
	*x = ListBarsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_service_bar_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListBarsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBarsResponse) ProtoMessage() {}

func (x *ListBarsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_service_bar_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBarsResponse.ProtoReflect.Descriptor instead.
func (*ListBarsResponse) Descriptor() ([]byte, []int) {
	return file_test_v1_service_bar_proto_rawDescGZIP(), []int{3}
}

func (x *ListBarsResponse) GetBars() []*test_pb.BarState {
	if x != nil {
		return x.Bars
	}
	return nil
}

func (x *ListBarsResponse) GetPage() *list_j5pb.PageResponse {
	if x != nil {
		return x.Page
	}
	return nil
}

type ListBarEventsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BarId string                  `protobuf:"bytes,1,opt,name=bar_id,json=barId,proto3" json:"bar_id,omitempty"`
	Page  *list_j5pb.PageRequest  `protobuf:"bytes,2,opt,name=page,proto3" json:"page,omitempty"`
	Query *list_j5pb.QueryRequest `protobuf:"bytes,3,opt,name=query,proto3" json:"query,omitempty"`
}

func (x *ListBarEventsRequest) Reset() {
	*x = ListBarEventsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_service_bar_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListBarEventsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBarEventsRequest) ProtoMessage() {}

func (x *ListBarEventsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_service_bar_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBarEventsRequest.ProtoReflect.Descriptor instead.
func (*ListBarEventsRequest) Descriptor() ([]byte, []int) {
	return file_test_v1_service_bar_proto_rawDescGZIP(), []int{4}
}

func (x *ListBarEventsRequest) GetBarId() string {
	if x != nil {
		return x.BarId
	}
	return ""
}

func (x *ListBarEventsRequest) GetPage() *list_j5pb.PageRequest {
	if x != nil {
		return x.Page
	}
	return nil
}

func (x *ListBarEventsRequest) GetQuery() *list_j5pb.QueryRequest {
	if x != nil {
		return x.Query
	}
	return nil
}

type ListBarEventsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Events []*test_pb.BarEvent     `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
	Page   *list_j5pb.PageResponse `protobuf:"bytes,2,opt,name=page,proto3" json:"page,omitempty"`
}

func (x *ListBarEventsResponse) Reset() {
	*x = ListBarEventsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_v1_service_bar_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListBarEventsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBarEventsResponse) ProtoMessage() {}

func (x *ListBarEventsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_test_v1_service_bar_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBarEventsResponse.ProtoReflect.Descriptor instead.
func (*ListBarEventsResponse) Descriptor() ([]byte, []int) {
	return file_test_v1_service_bar_proto_rawDescGZIP(), []int{5}
}

func (x *ListBarEventsResponse) GetEvents() []*test_pb.BarEvent {
	if x != nil {
		return x.Events
	}
	return nil
}

func (x *ListBarEventsResponse) GetPage() *list_j5pb.PageResponse {
	if x != nil {
		return x.Page
	}
	return nil
}

var File_test_v1_service_bar_proto protoreflect.FileDescriptor

var file_test_v1_service_bar_proto_rawDesc = []byte{
	0x0a, 0x19, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x2f, 0x62, 0x61, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0f, 0x74, 0x65, 0x73,
	0x74, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x1b, 0x62, 0x75,
	0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64,
	0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x6a, 0x35, 0x2f, 0x65, 0x78, 0x74, 0x2f,
	0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x6a, 0x35, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76, 0x31,
	0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x15, 0x6a, 0x35, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x70,
	0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x16, 0x6a, 0x35, 0x2f, 0x6c, 0x69,
	0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x11, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x62, 0x61, 0x72, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x30, 0x0a, 0x0d, 0x47, 0x65, 0x74, 0x42, 0x61, 0x72, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x06, 0x62, 0x61, 0x72, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x52,
	0x05, 0x62, 0x61, 0x72, 0x49, 0x64, 0x22, 0x64, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x42, 0x61, 0x72,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x27, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76,
	0x31, 0x2e, 0x42, 0x61, 0x72, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x12, 0x29, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x11, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x61, 0x72, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x22, 0xbd, 0x01, 0x0a,
	0x0f, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x61, 0x72, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x2a, 0x0a, 0x09, 0x74, 0x65, 0x6e, 0x61, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x48, 0x00, 0x52,
	0x08, 0x74, 0x65, 0x6e, 0x61, 0x6e, 0x74, 0x49, 0x64, 0x88, 0x01, 0x01, 0x12, 0x2b, 0x0a, 0x04,
	0x70, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x6a, 0x35, 0x2e,
	0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x52, 0x04, 0x70, 0x61, 0x67, 0x65, 0x12, 0x2e, 0x0a, 0x05, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x6a, 0x35, 0x2e, 0x6c, 0x69,
	0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x52, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x3a, 0x13, 0x8a, 0xf7, 0x98, 0xc6, 0x02,
	0x0d, 0x12, 0x0b, 0x6b, 0x65, 0x79, 0x73, 0x2e, 0x62, 0x61, 0x72, 0x5f, 0x69, 0x64, 0x42, 0x0c,
	0x0a, 0x0a, 0x5f, 0x74, 0x65, 0x6e, 0x61, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x22, 0x67, 0x0a, 0x10,
	0x4c, 0x69, 0x73, 0x74, 0x42, 0x61, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x25, 0x0a, 0x04, 0x62, 0x61, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11,
	0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x61, 0x72, 0x53, 0x74, 0x61, 0x74,
	0x65, 0x52, 0x04, 0x62, 0x61, 0x72, 0x73, 0x12, 0x2c, 0x0a, 0x04, 0x70, 0x61, 0x67, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x6a, 0x35, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e,
	0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x52,
	0x04, 0x70, 0x61, 0x67, 0x65, 0x22, 0x94, 0x01, 0x0a, 0x14, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x61,
	0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1f,
	0x0a, 0x06, 0x62, 0x61, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08,
	0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x52, 0x05, 0x62, 0x61, 0x72, 0x49, 0x64, 0x12,
	0x2b, 0x0a, 0x04, 0x70, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e,
	0x6a, 0x35, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x04, 0x70, 0x61, 0x67, 0x65, 0x12, 0x2e, 0x0a, 0x05,
	0x71, 0x75, 0x65, 0x72, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x6a, 0x35,
	0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x22, 0x70, 0x0a, 0x15,
	0x4c, 0x69, 0x73, 0x74, 0x42, 0x61, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x29, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e,
	0x42, 0x61, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73,
	0x12, 0x2c, 0x0a, 0x04, 0x70, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18,
	0x2e, 0x6a, 0x35, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x52, 0x04, 0x70, 0x61, 0x67, 0x65, 0x32, 0x8e,
	0x03, 0x0a, 0x0a, 0x42, 0x61, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x71, 0x0a,
	0x06, 0x47, 0x65, 0x74, 0x42, 0x61, 0x72, 0x12, 0x1e, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76,
	0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x42, 0x61, 0x72,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1f, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76,
	0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x42, 0x61, 0x72,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x26, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x52,
	0x02, 0x08, 0x01, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x17, 0x12, 0x15, 0x2f, 0x74, 0x65, 0x73, 0x74,
	0x2f, 0x76, 0x31, 0x2f, 0x62, 0x61, 0x72, 0x2f, 0x7b, 0x62, 0x61, 0x72, 0x5f, 0x69, 0x64, 0x7d,
	0x12, 0x6f, 0x0a, 0x08, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x61, 0x72, 0x73, 0x12, 0x20, 0x2e, 0x74,
	0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c,
	0x69, 0x73, 0x74, 0x42, 0x61, 0x72, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x21,
	0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x2e, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x61, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x1e, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x52, 0x02, 0x10, 0x01, 0x82, 0xd3, 0xe4, 0x93,
	0x02, 0x0f, 0x12, 0x0d, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x62, 0x61, 0x72,
	0x73, 0x12, 0x8d, 0x01, 0x0a, 0x0d, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x61, 0x72, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x73, 0x12, 0x25, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x61, 0x72, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x26, 0x2e, 0x74, 0x65, 0x73,
	0x74, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73,
	0x74, 0x42, 0x61, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x2d, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x52, 0x02, 0x18, 0x01, 0x82, 0xd3, 0xe4,
	0x93, 0x02, 0x1e, 0x12, 0x1c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x62, 0x61,
	0x72, 0x2f, 0x7b, 0x62, 0x61, 0x72, 0x5f, 0x69, 0x64, 0x7d, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74,
	0x73, 0x1a, 0x0c, 0xea, 0x85, 0x8f, 0x02, 0x07, 0x0a, 0x05, 0x0a, 0x03, 0x62, 0x61, 0x72, 0x42,
	0x47, 0x5a, 0x45, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65,
	0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x74, 0x61, 0x74, 0x65,
	0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f,
	0x74, 0x65, 0x73, 0x74, 0x5f, 0x73, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_test_v1_service_bar_proto_rawDescOnce sync.Once
	file_test_v1_service_bar_proto_rawDescData = file_test_v1_service_bar_proto_rawDesc
)

func file_test_v1_service_bar_proto_rawDescGZIP() []byte {
	file_test_v1_service_bar_proto_rawDescOnce.Do(func() {
		file_test_v1_service_bar_proto_rawDescData = protoimpl.X.CompressGZIP(file_test_v1_service_bar_proto_rawDescData)
	})
	return file_test_v1_service_bar_proto_rawDescData
}

var file_test_v1_service_bar_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_test_v1_service_bar_proto_goTypes = []interface{}{
	(*GetBarRequest)(nil),          // 0: test.v1.service.GetBarRequest
	(*GetBarResponse)(nil),         // 1: test.v1.service.GetBarResponse
	(*ListBarsRequest)(nil),        // 2: test.v1.service.ListBarsRequest
	(*ListBarsResponse)(nil),       // 3: test.v1.service.ListBarsResponse
	(*ListBarEventsRequest)(nil),   // 4: test.v1.service.ListBarEventsRequest
	(*ListBarEventsResponse)(nil),  // 5: test.v1.service.ListBarEventsResponse
	(*test_pb.BarState)(nil),       // 6: test.v1.BarState
	(*test_pb.BarEvent)(nil),       // 7: test.v1.BarEvent
	(*list_j5pb.PageRequest)(nil),  // 8: j5.list.v1.PageRequest
	(*list_j5pb.QueryRequest)(nil), // 9: j5.list.v1.QueryRequest
	(*list_j5pb.PageResponse)(nil), // 10: j5.list.v1.PageResponse
}
var file_test_v1_service_bar_proto_depIdxs = []int32{
	6,  // 0: test.v1.service.GetBarResponse.state:type_name -> test.v1.BarState
	7,  // 1: test.v1.service.GetBarResponse.events:type_name -> test.v1.BarEvent
	8,  // 2: test.v1.service.ListBarsRequest.page:type_name -> j5.list.v1.PageRequest
	9,  // 3: test.v1.service.ListBarsRequest.query:type_name -> j5.list.v1.QueryRequest
	6,  // 4: test.v1.service.ListBarsResponse.bars:type_name -> test.v1.BarState
	10, // 5: test.v1.service.ListBarsResponse.page:type_name -> j5.list.v1.PageResponse
	8,  // 6: test.v1.service.ListBarEventsRequest.page:type_name -> j5.list.v1.PageRequest
	9,  // 7: test.v1.service.ListBarEventsRequest.query:type_name -> j5.list.v1.QueryRequest
	7,  // 8: test.v1.service.ListBarEventsResponse.events:type_name -> test.v1.BarEvent
	10, // 9: test.v1.service.ListBarEventsResponse.page:type_name -> j5.list.v1.PageResponse
	0,  // 10: test.v1.service.BarService.GetBar:input_type -> test.v1.service.GetBarRequest
	2,  // 11: test.v1.service.BarService.ListBars:input_type -> test.v1.service.ListBarsRequest
	4,  // 12: test.v1.service.BarService.ListBarEvents:input_type -> test.v1.service.ListBarEventsRequest
	1,  // 13: test.v1.service.BarService.GetBar:output_type -> test.v1.service.GetBarResponse
	3,  // 14: test.v1.service.BarService.ListBars:output_type -> test.v1.service.ListBarsResponse
	5,  // 15: test.v1.service.BarService.ListBarEvents:output_type -> test.v1.service.ListBarEventsResponse
	13, // [13:16] is the sub-list for method output_type
	10, // [10:13] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_test_v1_service_bar_proto_init() }
func file_test_v1_service_bar_proto_init() {
	if File_test_v1_service_bar_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_test_v1_service_bar_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetBarRequest); i {
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
		file_test_v1_service_bar_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetBarResponse); i {
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
		file_test_v1_service_bar_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListBarsRequest); i {
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
		file_test_v1_service_bar_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListBarsResponse); i {
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
		file_test_v1_service_bar_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListBarEventsRequest); i {
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
		file_test_v1_service_bar_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListBarEventsResponse); i {
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
	file_test_v1_service_bar_proto_msgTypes[2].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_test_v1_service_bar_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_test_v1_service_bar_proto_goTypes,
		DependencyIndexes: file_test_v1_service_bar_proto_depIdxs,
		MessageInfos:      file_test_v1_service_bar_proto_msgTypes,
	}.Build()
	File_test_v1_service_bar_proto = out.File
	file_test_v1_service_bar_proto_rawDesc = nil
	file_test_v1_service_bar_proto_goTypes = nil
	file_test_v1_service_bar_proto_depIdxs = nil
}
