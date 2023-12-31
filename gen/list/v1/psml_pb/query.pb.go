// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: psm/list/v1/query.proto

package psml_pb

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
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

type QueryRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Search  []*Search `protobuf:"bytes,1,rep,name=search,proto3" json:"search,omitempty"`
	Sort    []*Sort   `protobuf:"bytes,2,rep,name=sort,proto3" json:"sort,omitempty"`
	Filters []*Filter `protobuf:"bytes,3,rep,name=filters,proto3" json:"filters,omitempty"`
}

func (x *QueryRequest) Reset() {
	*x = QueryRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_list_v1_query_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QueryRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QueryRequest) ProtoMessage() {}

func (x *QueryRequest) ProtoReflect() protoreflect.Message {
	mi := &file_psm_list_v1_query_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QueryRequest.ProtoReflect.Descriptor instead.
func (*QueryRequest) Descriptor() ([]byte, []int) {
	return file_psm_list_v1_query_proto_rawDescGZIP(), []int{0}
}

func (x *QueryRequest) GetSearch() []*Search {
	if x != nil {
		return x.Search
	}
	return nil
}

func (x *QueryRequest) GetSort() []*Sort {
	if x != nil {
		return x.Sort
	}
	return nil
}

func (x *QueryRequest) GetFilters() []*Filter {
	if x != nil {
		return x.Filters
	}
	return nil
}

type Search struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Field string `protobuf:"bytes,1,opt,name=field,proto3" json:"field,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *Search) Reset() {
	*x = Search{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_list_v1_query_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Search) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Search) ProtoMessage() {}

func (x *Search) ProtoReflect() protoreflect.Message {
	mi := &file_psm_list_v1_query_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Search.ProtoReflect.Descriptor instead.
func (*Search) Descriptor() ([]byte, []int) {
	return file_psm_list_v1_query_proto_rawDescGZIP(), []int{1}
}

func (x *Search) GetField() string {
	if x != nil {
		return x.Field
	}
	return ""
}

func (x *Search) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

type Sort struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Field      string `protobuf:"bytes,1,opt,name=field,proto3" json:"field,omitempty"`
	Descending bool   `protobuf:"varint,2,opt,name=descending,proto3" json:"descending,omitempty"`
}

func (x *Sort) Reset() {
	*x = Sort{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_list_v1_query_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Sort) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Sort) ProtoMessage() {}

func (x *Sort) ProtoReflect() protoreflect.Message {
	mi := &file_psm_list_v1_query_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Sort.ProtoReflect.Descriptor instead.
func (*Sort) Descriptor() ([]byte, []int) {
	return file_psm_list_v1_query_proto_rawDescGZIP(), []int{2}
}

func (x *Sort) GetField() string {
	if x != nil {
		return x.Field
	}
	return ""
}

func (x *Sort) GetDescending() bool {
	if x != nil {
		return x.Descending
	}
	return false
}

type Filter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Field string `protobuf:"bytes,1,opt,name=field,proto3" json:"field,omitempty"`
	// Types that are assignable to FilterValue:
	//
	//	*Filter_Value
	//	*Filter_Range
	FilterValue isFilter_FilterValue `protobuf_oneof:"filter_value"`
}

func (x *Filter) Reset() {
	*x = Filter{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_list_v1_query_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Filter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Filter) ProtoMessage() {}

func (x *Filter) ProtoReflect() protoreflect.Message {
	mi := &file_psm_list_v1_query_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Filter.ProtoReflect.Descriptor instead.
func (*Filter) Descriptor() ([]byte, []int) {
	return file_psm_list_v1_query_proto_rawDescGZIP(), []int{3}
}

func (x *Filter) GetField() string {
	if x != nil {
		return x.Field
	}
	return ""
}

func (m *Filter) GetFilterValue() isFilter_FilterValue {
	if m != nil {
		return m.FilterValue
	}
	return nil
}

func (x *Filter) GetValue() string {
	if x, ok := x.GetFilterValue().(*Filter_Value); ok {
		return x.Value
	}
	return ""
}

func (x *Filter) GetRange() *Range {
	if x, ok := x.GetFilterValue().(*Filter_Range); ok {
		return x.Range
	}
	return nil
}

type isFilter_FilterValue interface {
	isFilter_FilterValue()
}

type Filter_Value struct {
	Value string `protobuf:"bytes,2,opt,name=value,proto3,oneof"`
}

type Filter_Range struct {
	Range *Range `protobuf:"bytes,3,opt,name=range,proto3,oneof"`
}

func (*Filter_Value) isFilter_FilterValue() {}

func (*Filter_Range) isFilter_FilterValue() {}

type Range struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Min string `protobuf:"bytes,1,opt,name=min,proto3" json:"min,omitempty"`
	Max string `protobuf:"bytes,2,opt,name=max,proto3" json:"max,omitempty"`
}

func (x *Range) Reset() {
	*x = Range{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_list_v1_query_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Range) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Range) ProtoMessage() {}

func (x *Range) ProtoReflect() protoreflect.Message {
	mi := &file_psm_list_v1_query_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Range.ProtoReflect.Descriptor instead.
func (*Range) Descriptor() ([]byte, []int) {
	return file_psm_list_v1_query_proto_rawDescGZIP(), []int{4}
}

func (x *Range) GetMin() string {
	if x != nil {
		return x.Min
	}
	return ""
}

func (x *Range) GetMax() string {
	if x != nil {
		return x.Max
	}
	return ""
}

var File_psm_list_v1_query_proto protoreflect.FileDescriptor

var file_psm_list_v1_query_proto_rawDesc = []byte{
	0x0a, 0x17, 0x70, 0x73, 0x6d, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x71, 0x75,
	0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x70, 0x73, 0x6d, 0x2e, 0x6c,
	0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69,
	0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x91, 0x01, 0x0a, 0x0c, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x2b, 0x0a, 0x06, 0x73, 0x65, 0x61, 0x72, 0x63, 0x68, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e,
	0x76, 0x31, 0x2e, 0x53, 0x65, 0x61, 0x72, 0x63, 0x68, 0x52, 0x06, 0x73, 0x65, 0x61, 0x72, 0x63,
	0x68, 0x12, 0x25, 0x0a, 0x04, 0x73, 0x6f, 0x72, 0x74, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x11, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x6f,
	0x72, 0x74, 0x52, 0x04, 0x73, 0x6f, 0x72, 0x74, 0x12, 0x2d, 0x0a, 0x07, 0x66, 0x69, 0x6c, 0x74,
	0x65, 0x72, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x70, 0x73, 0x6d, 0x2e,
	0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x52, 0x07,
	0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x73, 0x22, 0x34, 0x0a, 0x06, 0x53, 0x65, 0x61, 0x72, 0x63,
	0x68, 0x12, 0x14, 0x0a, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x3c, 0x0a,
	0x04, 0x53, 0x6f, 0x72, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x1e, 0x0a, 0x0a, 0x64,
	0x65, 0x73, 0x63, 0x65, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x0a, 0x64, 0x65, 0x73, 0x63, 0x65, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x22, 0x79, 0x0a, 0x06, 0x46,
	0x69, 0x6c, 0x74, 0x65, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x16, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x12, 0x2a, 0x0a, 0x05, 0x72, 0x61, 0x6e, 0x67, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x12, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31,
	0x2e, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x48, 0x00, 0x52, 0x05, 0x72, 0x61, 0x6e, 0x67, 0x65, 0x42,
	0x15, 0x0a, 0x0c, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12,
	0x05, 0xba, 0x48, 0x02, 0x08, 0x01, 0x22, 0x2b, 0x0a, 0x05, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x12,
	0x10, 0x0a, 0x03, 0x6d, 0x69, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6d, 0x69,
	0x6e, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x61, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6d, 0x61, 0x78, 0x42, 0x33, 0x5a, 0x31, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73,
	0x74, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76, 0x31,
	0x2f, 0x70, 0x73, 0x6d, 0x6c, 0x5f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_psm_list_v1_query_proto_rawDescOnce sync.Once
	file_psm_list_v1_query_proto_rawDescData = file_psm_list_v1_query_proto_rawDesc
)

func file_psm_list_v1_query_proto_rawDescGZIP() []byte {
	file_psm_list_v1_query_proto_rawDescOnce.Do(func() {
		file_psm_list_v1_query_proto_rawDescData = protoimpl.X.CompressGZIP(file_psm_list_v1_query_proto_rawDescData)
	})
	return file_psm_list_v1_query_proto_rawDescData
}

var file_psm_list_v1_query_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_psm_list_v1_query_proto_goTypes = []interface{}{
	(*QueryRequest)(nil), // 0: psm.list.v1.QueryRequest
	(*Search)(nil),       // 1: psm.list.v1.Search
	(*Sort)(nil),         // 2: psm.list.v1.Sort
	(*Filter)(nil),       // 3: psm.list.v1.Filter
	(*Range)(nil),        // 4: psm.list.v1.Range
}
var file_psm_list_v1_query_proto_depIdxs = []int32{
	1, // 0: psm.list.v1.QueryRequest.search:type_name -> psm.list.v1.Search
	2, // 1: psm.list.v1.QueryRequest.sort:type_name -> psm.list.v1.Sort
	3, // 2: psm.list.v1.QueryRequest.filters:type_name -> psm.list.v1.Filter
	4, // 3: psm.list.v1.Filter.range:type_name -> psm.list.v1.Range
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_psm_list_v1_query_proto_init() }
func file_psm_list_v1_query_proto_init() {
	if File_psm_list_v1_query_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_psm_list_v1_query_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QueryRequest); i {
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
		file_psm_list_v1_query_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Search); i {
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
		file_psm_list_v1_query_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Sort); i {
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
		file_psm_list_v1_query_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Filter); i {
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
		file_psm_list_v1_query_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Range); i {
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
	file_psm_list_v1_query_proto_msgTypes[3].OneofWrappers = []interface{}{
		(*Filter_Value)(nil),
		(*Filter_Range)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_psm_list_v1_query_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_psm_list_v1_query_proto_goTypes,
		DependencyIndexes: file_psm_list_v1_query_proto_depIdxs,
		MessageInfos:      file_psm_list_v1_query_proto_msgTypes,
	}.Build()
	File_psm_list_v1_query_proto = out.File
	file_psm_list_v1_query_proto_rawDesc = nil
	file_psm_list_v1_query_proto_goTypes = nil
	file_psm_list_v1_query_proto_depIdxs = nil
}
