// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.3
// 	protoc        buf-v1.55.1
// source: sample/v1/model.proto

package samplev1

import (
	reflect "reflect"
	sync "sync"

	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Reachability test response.
type Pong struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Whether the service is 'ready-for-use'
	Ok            bool `protobuf:"varint,1,opt,name=ok,proto3" json:"ok,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Pong) Reset() {
	*x = Pong{}
	mi := &file_sample_v1_model_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Pong) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Pong) ProtoMessage() {}

func (x *Pong) ProtoReflect() protoreflect.Message {
	mi := &file_sample_v1_model_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Pong.ProtoReflect.Descriptor instead.
func (*Pong) Descriptor() ([]byte, []int) {
	return file_sample_v1_model_proto_rawDescGZIP(), []int{0}
}

func (x *Pong) GetOk() bool {
	if x != nil {
		return x.Ok
	}
	return false
}

// Health test response.
type HealthResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Generic availability check, set to 'false' in case of errors.
	Alive         bool `protobuf:"varint,1,opt,name=alive,proto3" json:"alive,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HealthResponse) Reset() {
	*x = HealthResponse{}
	mi := &file_sample_v1_model_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HealthResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HealthResponse) ProtoMessage() {}

func (x *HealthResponse) ProtoReflect() protoreflect.Message {
	mi := &file_sample_v1_model_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HealthResponse.ProtoReflect.Descriptor instead.
func (*HealthResponse) Descriptor() ([]byte, []int) {
	return file_sample_v1_model_proto_rawDescGZIP(), []int{1}
}

func (x *HealthResponse) GetAlive() bool {
	if x != nil {
		return x.Alive
	}
	return false
}

// Generic request response.
type Response struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// A key on the entity
	Name          string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Response) Reset() {
	*x = Response{}
	mi := &file_sample_v1_model_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Response) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response) ProtoMessage() {}

func (x *Response) ProtoReflect() protoreflect.Message {
	mi := &file_sample_v1_model_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response.ProtoReflect.Descriptor instead.
func (*Response) Descriptor() ([]byte, []int) {
	return file_sample_v1_model_proto_rawDescGZIP(), []int{2}
}

func (x *Response) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

// Sample response structure.
type DummyResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Generic status check, set to 'false' in case of errors.
	Ok            bool `protobuf:"varint,1,opt,name=ok,proto3" json:"ok,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DummyResponse) Reset() {
	*x = DummyResponse{}
	mi := &file_sample_v1_model_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DummyResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DummyResponse) ProtoMessage() {}

func (x *DummyResponse) ProtoReflect() protoreflect.Message {
	mi := &file_sample_v1_model_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DummyResponse.ProtoReflect.Descriptor instead.
func (*DummyResponse) Descriptor() ([]byte, []int) {
	return file_sample_v1_model_proto_rawDescGZIP(), []int{3}
}

func (x *DummyResponse) GetOk() bool {
	if x != nil {
		return x.Ok
	}
	return false
}

var File_sample_v1_model_proto protoreflect.FileDescriptor

var file_sample_v1_model_proto_rawDesc = []byte{
	0x0a, 0x15, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x6f, 0x64, 0x65,
	0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e,
	0x76, 0x31, 0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65,
	0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x16, 0x0a, 0x04, 0x50, 0x6f, 0x6e, 0x67, 0x12, 0x0e, 0x0a, 0x02, 0x6f, 0x6b, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x02, 0x6f, 0x6b, 0x22, 0x26, 0x0a, 0x0e, 0x48, 0x65, 0x61, 0x6c, 0x74,
	0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x61, 0x6c, 0x69,
	0x76, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x61, 0x6c, 0x69, 0x76, 0x65, 0x22,
	0x29, 0x0a, 0x08, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1d, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x09, 0xba, 0x48, 0x06, 0x72, 0x04,
	0x10, 0x02, 0x18, 0x05, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x1f, 0x0a, 0x0d, 0x44, 0x75,
	0x6d, 0x6d, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x6f,
	0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x02, 0x6f, 0x6b, 0x42, 0x83, 0x01, 0x0a, 0x0d,
	0x63, 0x6f, 0x6d, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x42, 0x0a, 0x4d,
	0x6f, 0x64, 0x65, 0x6c, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x21, 0x67, 0x6f, 0x2e,
	0x62, 0x72, 0x79, 0x6b, 0x2e, 0x69, 0x6f, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x73, 0x61, 0x6d, 0x70,
	0x6c, 0x65, 0x2f, 0x76, 0x31, 0x3b, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x76, 0x31, 0xa2, 0x02,
	0x03, 0x53, 0x58, 0x58, 0xaa, 0x02, 0x09, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x56, 0x31,
	0xca, 0x02, 0x09, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x15, 0x53,
	0x61, 0x6d, 0x70, 0x6c, 0x65, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0a, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x3a, 0x3a, 0x56,
	0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_sample_v1_model_proto_rawDescOnce sync.Once
	file_sample_v1_model_proto_rawDescData = file_sample_v1_model_proto_rawDesc
)

func file_sample_v1_model_proto_rawDescGZIP() []byte {
	file_sample_v1_model_proto_rawDescOnce.Do(func() {
		file_sample_v1_model_proto_rawDescData = protoimpl.X.CompressGZIP(file_sample_v1_model_proto_rawDescData)
	})
	return file_sample_v1_model_proto_rawDescData
}

var file_sample_v1_model_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_sample_v1_model_proto_goTypes = []any{
	(*Pong)(nil),           // 0: sample.v1.Pong
	(*HealthResponse)(nil), // 1: sample.v1.HealthResponse
	(*Response)(nil),       // 2: sample.v1.Response
	(*DummyResponse)(nil),  // 3: sample.v1.DummyResponse
}
var file_sample_v1_model_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_sample_v1_model_proto_init() }
func file_sample_v1_model_proto_init() {
	if File_sample_v1_model_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_sample_v1_model_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_sample_v1_model_proto_goTypes,
		DependencyIndexes: file_sample_v1_model_proto_depIdxs,
		MessageInfos:      file_sample_v1_model_proto_msgTypes,
	}.Build()
	File_sample_v1_model_proto = out.File
	file_sample_v1_model_proto_rawDesc = nil
	file_sample_v1_model_proto_goTypes = nil
	file_sample_v1_model_proto_depIdxs = nil
}
