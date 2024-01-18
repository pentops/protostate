// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: psm/state/v1/annotations.proto

package psm_pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type StateObjectOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *StateObjectOptions) Reset() {
	*x = StateObjectOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_state_v1_annotations_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StateObjectOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StateObjectOptions) ProtoMessage() {}

func (x *StateObjectOptions) ProtoReflect() protoreflect.Message {
	mi := &file_psm_state_v1_annotations_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StateObjectOptions.ProtoReflect.Descriptor instead.
func (*StateObjectOptions) Descriptor() ([]byte, []int) {
	return file_psm_state_v1_annotations_proto_rawDescGZIP(), []int{0}
}

func (x *StateObjectOptions) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type EventObjectOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *EventObjectOptions) Reset() {
	*x = EventObjectOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_state_v1_annotations_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventObjectOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventObjectOptions) ProtoMessage() {}

func (x *EventObjectOptions) ProtoReflect() protoreflect.Message {
	mi := &file_psm_state_v1_annotations_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventObjectOptions.ProtoReflect.Descriptor instead.
func (*EventObjectOptions) Descriptor() ([]byte, []int) {
	return file_psm_state_v1_annotations_proto_rawDescGZIP(), []int{1}
}

func (x *EventObjectOptions) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type EventTypeObjectOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *EventTypeObjectOptions) Reset() {
	*x = EventTypeObjectOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_state_v1_annotations_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventTypeObjectOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventTypeObjectOptions) ProtoMessage() {}

func (x *EventTypeObjectOptions) ProtoReflect() protoreflect.Message {
	mi := &file_psm_state_v1_annotations_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventTypeObjectOptions.ProtoReflect.Descriptor instead.
func (*EventTypeObjectOptions) Descriptor() ([]byte, []int) {
	return file_psm_state_v1_annotations_proto_rawDescGZIP(), []int{2}
}

type StateField struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// the field is the primary key, or is one field in a multi-key primary key
	// for the state object.
	PrimaryKey bool `protobuf:"varint,1,opt,name=primary_key,json=primaryKey,proto3" json:"primary_key,omitempty"`
}

func (x *StateField) Reset() {
	*x = StateField{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_state_v1_annotations_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StateField) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StateField) ProtoMessage() {}

func (x *StateField) ProtoReflect() protoreflect.Message {
	mi := &file_psm_state_v1_annotations_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StateField.ProtoReflect.Descriptor instead.
func (*StateField) Descriptor() ([]byte, []int) {
	return file_psm_state_v1_annotations_proto_rawDescGZIP(), []int{3}
}

func (x *StateField) GetPrimaryKey() bool {
	if x != nil {
		return x.PrimaryKey
	}
	return false
}

type EventField struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// the field is the enum wrapper for the event type
	EventType bool `protobuf:"varint,1,opt,name=event_type,json=eventType,proto3" json:"event_type,omitempty"`
	// the field contains the metadata for this message
	Metadata bool `protobuf:"varint,2,opt,name=metadata,proto3" json:"metadata,omitempty"`
	// the field contains a foreign key to the state object. This must be part of
	// the state's primary key, i.e. the state's field must be marked as
	// primary_key = true;
	StateKey bool `protobuf:"varint,3,opt,name=state_key,json=stateKey,proto3" json:"state_key,omitempty"`
	// a non-primary key field used to construct initial state. When this is
	// specified on an event for a state which already exists, it must equal the
	// value in the state after the transition caused by the event.
	StateField bool `protobuf:"varint,4,opt,name=state_field,json=stateField,proto3" json:"state_field,omitempty"`
}

func (x *EventField) Reset() {
	*x = EventField{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_state_v1_annotations_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventField) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventField) ProtoMessage() {}

func (x *EventField) ProtoReflect() protoreflect.Message {
	mi := &file_psm_state_v1_annotations_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventField.ProtoReflect.Descriptor instead.
func (*EventField) Descriptor() ([]byte, []int) {
	return file_psm_state_v1_annotations_proto_rawDescGZIP(), []int{4}
}

func (x *EventField) GetEventType() bool {
	if x != nil {
		return x.EventType
	}
	return false
}

func (x *EventField) GetMetadata() bool {
	if x != nil {
		return x.Metadata
	}
	return false
}

func (x *EventField) GetStateKey() bool {
	if x != nil {
		return x.StateKey
	}
	return false
}

func (x *EventField) GetStateField() bool {
	if x != nil {
		return x.StateField
	}
	return false
}

type EventTypeField struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SubKey bool `protobuf:"varint,1,opt,name=sub_key,json=subKey,proto3" json:"sub_key,omitempty"`
}

func (x *EventTypeField) Reset() {
	*x = EventTypeField{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_state_v1_annotations_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EventTypeField) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EventTypeField) ProtoMessage() {}

func (x *EventTypeField) ProtoReflect() protoreflect.Message {
	mi := &file_psm_state_v1_annotations_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EventTypeField.ProtoReflect.Descriptor instead.
func (*EventTypeField) Descriptor() ([]byte, []int) {
	return file_psm_state_v1_annotations_proto_rawDescGZIP(), []int{5}
}

func (x *EventTypeField) GetSubKey() bool {
	if x != nil {
		return x.SubKey
	}
	return false
}

type StateQueryServiceOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *StateQueryServiceOptions) Reset() {
	*x = StateQueryServiceOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_state_v1_annotations_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StateQueryServiceOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StateQueryServiceOptions) ProtoMessage() {}

func (x *StateQueryServiceOptions) ProtoReflect() protoreflect.Message {
	mi := &file_psm_state_v1_annotations_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StateQueryServiceOptions.ProtoReflect.Descriptor instead.
func (*StateQueryServiceOptions) Descriptor() ([]byte, []int) {
	return file_psm_state_v1_annotations_proto_rawDescGZIP(), []int{6}
}

func (x *StateQueryServiceOptions) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type StateQueryMethodOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Get        bool `protobuf:"varint,1,opt,name=get,proto3" json:"get,omitempty"`
	List       bool `protobuf:"varint,2,opt,name=list,proto3" json:"list,omitempty"`
	ListEvents bool `protobuf:"varint,3,opt,name=list_events,json=listEvents,proto3" json:"list_events,omitempty"`
	// use name here instead of the service option to put multiple state query
	// methods in one service
	Name string `protobuf:"bytes,4,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *StateQueryMethodOptions) Reset() {
	*x = StateQueryMethodOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_psm_state_v1_annotations_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StateQueryMethodOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StateQueryMethodOptions) ProtoMessage() {}

func (x *StateQueryMethodOptions) ProtoReflect() protoreflect.Message {
	mi := &file_psm_state_v1_annotations_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StateQueryMethodOptions.ProtoReflect.Descriptor instead.
func (*StateQueryMethodOptions) Descriptor() ([]byte, []int) {
	return file_psm_state_v1_annotations_proto_rawDescGZIP(), []int{7}
}

func (x *StateQueryMethodOptions) GetGet() bool {
	if x != nil {
		return x.Get
	}
	return false
}

func (x *StateQueryMethodOptions) GetList() bool {
	if x != nil {
		return x.List
	}
	return false
}

func (x *StateQueryMethodOptions) GetListEvents() bool {
	if x != nil {
		return x.ListEvents
	}
	return false
}

func (x *StateQueryMethodOptions) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

var file_psm_state_v1_annotations_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptorpb.MessageOptions)(nil),
		ExtensionType: (*StateObjectOptions)(nil),
		Field:         90443400,
		Name:          "psm.state.v1.state",
		Tag:           "bytes,90443400,opt,name=state",
		Filename:      "psm/state/v1/annotations.proto",
	},
	{
		ExtendedType:  (*descriptorpb.MessageOptions)(nil),
		ExtensionType: (*EventObjectOptions)(nil),
		Field:         90443401,
		Name:          "psm.state.v1.event",
		Tag:           "bytes,90443401,opt,name=event",
		Filename:      "psm/state/v1/annotations.proto",
	},
	{
		ExtendedType:  (*descriptorpb.MessageOptions)(nil),
		ExtensionType: (*EventTypeObjectOptions)(nil),
		Field:         90443402,
		Name:          "psm.state.v1.event_type",
		Tag:           "bytes,90443402,opt,name=event_type",
		Filename:      "psm/state/v1/annotations.proto",
	},
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: (*StateField)(nil),
		Field:         90443405,
		Name:          "psm.state.v1.state_field",
		Tag:           "bytes,90443405,opt,name=state_field",
		Filename:      "psm/state/v1/annotations.proto",
	},
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: (*EventField)(nil),
		Field:         90443403,
		Name:          "psm.state.v1.event_field",
		Tag:           "bytes,90443403,opt,name=event_field",
		Filename:      "psm/state/v1/annotations.proto",
	},
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: (*EventTypeField)(nil),
		Field:         90443404,
		Name:          "psm.state.v1.event_type_field",
		Tag:           "bytes,90443404,opt,name=event_type_field",
		Filename:      "psm/state/v1/annotations.proto",
	},
	{
		ExtendedType:  (*descriptorpb.ServiceOptions)(nil),
		ExtensionType: (*StateQueryServiceOptions)(nil),
		Field:         90443405,
		Name:          "psm.state.v1.state_query",
		Tag:           "bytes,90443405,opt,name=state_query",
		Filename:      "psm/state/v1/annotations.proto",
	},
	{
		ExtendedType:  (*descriptorpb.MethodOptions)(nil),
		ExtensionType: (*StateQueryMethodOptions)(nil),
		Field:         90443406,
		Name:          "psm.state.v1.state_query_method",
		Tag:           "bytes,90443406,opt,name=state_query_method",
		Filename:      "psm/state/v1/annotations.proto",
	},
}

// Extension fields to descriptorpb.MessageOptions.
var (
	// optional psm.state.v1.StateObjectOptions state = 90443400;
	E_State = &file_psm_state_v1_annotations_proto_extTypes[0]
	// optional psm.state.v1.EventObjectOptions event = 90443401;
	E_Event = &file_psm_state_v1_annotations_proto_extTypes[1]
	// optional psm.state.v1.EventTypeObjectOptions event_type = 90443402;
	E_EventType = &file_psm_state_v1_annotations_proto_extTypes[2]
)

// Extension fields to descriptorpb.FieldOptions.
var (
	// optional psm.state.v1.StateField state_field = 90443405;
	E_StateField = &file_psm_state_v1_annotations_proto_extTypes[3]
	// optional psm.state.v1.EventField event_field = 90443403;
	E_EventField = &file_psm_state_v1_annotations_proto_extTypes[4]
	// optional psm.state.v1.EventTypeField event_type_field = 90443404;
	E_EventTypeField = &file_psm_state_v1_annotations_proto_extTypes[5]
)

// Extension fields to descriptorpb.ServiceOptions.
var (
	// optional psm.state.v1.StateQueryServiceOptions state_query = 90443405;
	E_StateQuery = &file_psm_state_v1_annotations_proto_extTypes[6]
)

// Extension fields to descriptorpb.MethodOptions.
var (
	// optional psm.state.v1.StateQueryMethodOptions state_query_method = 90443406;
	E_StateQueryMethod = &file_psm_state_v1_annotations_proto_extTypes[7]
)

var File_psm_state_v1_annotations_proto protoreflect.FileDescriptor

var file_psm_state_v1_annotations_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x70, 0x73, 0x6d, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x61,
	0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0c, 0x70, 0x73, 0x6d, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x1a, 0x20,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f,
	0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0x28, 0x0a, 0x12, 0x53, 0x74, 0x61, 0x74, 0x65, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x4f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x28, 0x0a, 0x12, 0x45, 0x76,
	0x65, 0x6e, 0x74, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x22, 0x18, 0x0a, 0x16, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70,
	0x65, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0x2d,
	0x0a, 0x0a, 0x53, 0x74, 0x61, 0x74, 0x65, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x1f, 0x0a, 0x0b,
	0x70, 0x72, 0x69, 0x6d, 0x61, 0x72, 0x79, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x0a, 0x70, 0x72, 0x69, 0x6d, 0x61, 0x72, 0x79, 0x4b, 0x65, 0x79, 0x22, 0x85, 0x01,
	0x0a, 0x0a, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x1d, 0x0a, 0x0a,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x09, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x6d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x6d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x1b, 0x0a, 0x09, 0x73, 0x74, 0x61, 0x74, 0x65,
	0x5f, 0x6b, 0x65, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x4b, 0x65, 0x79, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x66, 0x69,
	0x65, 0x6c, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x73, 0x74, 0x61, 0x74, 0x65,
	0x46, 0x69, 0x65, 0x6c, 0x64, 0x22, 0x29, 0x0a, 0x0e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79,
	0x70, 0x65, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x73, 0x75, 0x62, 0x5f, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x73, 0x75, 0x62, 0x4b, 0x65, 0x79,
	0x22, 0x2e, 0x0a, 0x18, 0x53, 0x74, 0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x12, 0x0a, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x22, 0x74, 0x0a, 0x17, 0x53, 0x74, 0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x4d, 0x65,
	0x74, 0x68, 0x6f, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x10, 0x0a, 0x03, 0x67,
	0x65, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x03, 0x67, 0x65, 0x74, 0x12, 0x12, 0x0a,
	0x04, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x6c, 0x69, 0x73,
	0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x6c, 0x69, 0x73, 0x74, 0x5f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x6c, 0x69, 0x73, 0x74, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x3a, 0x5a, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12,
	0x1f, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x18, 0x88, 0x9d, 0x90, 0x2b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x70, 0x73, 0x6d, 0x2e,
	0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x4f, 0x62,
	0x6a, 0x65, 0x63, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x05, 0x73, 0x74, 0x61,
	0x74, 0x65, 0x3a, 0x5a, 0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x1f, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x89, 0x9d, 0x90,
	0x2b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x3a, 0x67,
	0x0a, 0x0a, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x12, 0x1f, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x8a, 0x9d,
	0x90, 0x2b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x73, 0x74, 0x61,
	0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x4f,
	0x62, 0x6a, 0x65, 0x63, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x09, 0x65, 0x76,
	0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x3a, 0x5b, 0x0a, 0x0b, 0x73, 0x74, 0x61, 0x74, 0x65,
	0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x8d, 0x9d, 0x90, 0x2b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18,
	0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x65, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x52, 0x0a, 0x73, 0x74, 0x61, 0x74, 0x65, 0x46,
	0x69, 0x65, 0x6c, 0x64, 0x3a, 0x5b, 0x0a, 0x0b, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x66, 0x69,
	0x65, 0x6c, 0x64, 0x12, 0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x18, 0x8b, 0x9d, 0x90, 0x2b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x70, 0x73,
	0x6d, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74,
	0x46, 0x69, 0x65, 0x6c, 0x64, 0x52, 0x0a, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x46, 0x69, 0x65, 0x6c,
	0x64, 0x3a, 0x68, 0x0a, 0x10, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x5f,
	0x66, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x18, 0x8c, 0x9d, 0x90, 0x2b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e,
	0x70, 0x73, 0x6d, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65,
	0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x52, 0x0e, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x3a, 0x6b, 0x0a, 0x0b, 0x73,
	0x74, 0x61, 0x74, 0x65, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x12, 0x1f, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x8d, 0x9d, 0x90, 0x2b,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65,
	0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x0a, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x3a, 0x76, 0x0a, 0x12, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x1e,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x8e,
	0x9d, 0x90, 0x2b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72,
	0x79, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x10,
	0x73, 0x74, 0x61, 0x74, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64,
	0x42, 0x33, 0x5a, 0x31, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70,
	0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x70,
	0x73, 0x6d, 0x5f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_psm_state_v1_annotations_proto_rawDescOnce sync.Once
	file_psm_state_v1_annotations_proto_rawDescData = file_psm_state_v1_annotations_proto_rawDesc
)

func file_psm_state_v1_annotations_proto_rawDescGZIP() []byte {
	file_psm_state_v1_annotations_proto_rawDescOnce.Do(func() {
		file_psm_state_v1_annotations_proto_rawDescData = protoimpl.X.CompressGZIP(file_psm_state_v1_annotations_proto_rawDescData)
	})
	return file_psm_state_v1_annotations_proto_rawDescData
}

var file_psm_state_v1_annotations_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_psm_state_v1_annotations_proto_goTypes = []interface{}{
	(*StateObjectOptions)(nil),          // 0: psm.state.v1.StateObjectOptions
	(*EventObjectOptions)(nil),          // 1: psm.state.v1.EventObjectOptions
	(*EventTypeObjectOptions)(nil),      // 2: psm.state.v1.EventTypeObjectOptions
	(*StateField)(nil),                  // 3: psm.state.v1.StateField
	(*EventField)(nil),                  // 4: psm.state.v1.EventField
	(*EventTypeField)(nil),              // 5: psm.state.v1.EventTypeField
	(*StateQueryServiceOptions)(nil),    // 6: psm.state.v1.StateQueryServiceOptions
	(*StateQueryMethodOptions)(nil),     // 7: psm.state.v1.StateQueryMethodOptions
	(*descriptorpb.MessageOptions)(nil), // 8: google.protobuf.MessageOptions
	(*descriptorpb.FieldOptions)(nil),   // 9: google.protobuf.FieldOptions
	(*descriptorpb.ServiceOptions)(nil), // 10: google.protobuf.ServiceOptions
	(*descriptorpb.MethodOptions)(nil),  // 11: google.protobuf.MethodOptions
}
var file_psm_state_v1_annotations_proto_depIdxs = []int32{
	8,  // 0: psm.state.v1.state:extendee -> google.protobuf.MessageOptions
	8,  // 1: psm.state.v1.event:extendee -> google.protobuf.MessageOptions
	8,  // 2: psm.state.v1.event_type:extendee -> google.protobuf.MessageOptions
	9,  // 3: psm.state.v1.state_field:extendee -> google.protobuf.FieldOptions
	9,  // 4: psm.state.v1.event_field:extendee -> google.protobuf.FieldOptions
	9,  // 5: psm.state.v1.event_type_field:extendee -> google.protobuf.FieldOptions
	10, // 6: psm.state.v1.state_query:extendee -> google.protobuf.ServiceOptions
	11, // 7: psm.state.v1.state_query_method:extendee -> google.protobuf.MethodOptions
	0,  // 8: psm.state.v1.state:type_name -> psm.state.v1.StateObjectOptions
	1,  // 9: psm.state.v1.event:type_name -> psm.state.v1.EventObjectOptions
	2,  // 10: psm.state.v1.event_type:type_name -> psm.state.v1.EventTypeObjectOptions
	3,  // 11: psm.state.v1.state_field:type_name -> psm.state.v1.StateField
	4,  // 12: psm.state.v1.event_field:type_name -> psm.state.v1.EventField
	5,  // 13: psm.state.v1.event_type_field:type_name -> psm.state.v1.EventTypeField
	6,  // 14: psm.state.v1.state_query:type_name -> psm.state.v1.StateQueryServiceOptions
	7,  // 15: psm.state.v1.state_query_method:type_name -> psm.state.v1.StateQueryMethodOptions
	16, // [16:16] is the sub-list for method output_type
	16, // [16:16] is the sub-list for method input_type
	8,  // [8:16] is the sub-list for extension type_name
	0,  // [0:8] is the sub-list for extension extendee
	0,  // [0:0] is the sub-list for field type_name
}

func init() { file_psm_state_v1_annotations_proto_init() }
func file_psm_state_v1_annotations_proto_init() {
	if File_psm_state_v1_annotations_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_psm_state_v1_annotations_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StateObjectOptions); i {
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
		file_psm_state_v1_annotations_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventObjectOptions); i {
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
		file_psm_state_v1_annotations_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventTypeObjectOptions); i {
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
		file_psm_state_v1_annotations_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StateField); i {
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
		file_psm_state_v1_annotations_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventField); i {
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
		file_psm_state_v1_annotations_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EventTypeField); i {
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
		file_psm_state_v1_annotations_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StateQueryServiceOptions); i {
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
		file_psm_state_v1_annotations_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StateQueryMethodOptions); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_psm_state_v1_annotations_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 8,
			NumServices:   0,
		},
		GoTypes:           file_psm_state_v1_annotations_proto_goTypes,
		DependencyIndexes: file_psm_state_v1_annotations_proto_depIdxs,
		MessageInfos:      file_psm_state_v1_annotations_proto_msgTypes,
		ExtensionInfos:    file_psm_state_v1_annotations_proto_extTypes,
	}.Build()
	File_psm_state_v1_annotations_proto = out.File
	file_psm_state_v1_annotations_proto_rawDesc = nil
	file_psm_state_v1_annotations_proto_goTypes = nil
	file_psm_state_v1_annotations_proto_depIdxs = nil
}
