// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.1
// 	protoc        buf-v1.31.0
// source: sample/v1/foo_api.proto

package samplev1

import (
	reflect "reflect"
	sync "sync"

	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Generic stream messages returned by the server.
type GenericStreamChunk struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Identifier for the endpoint generating the stream message.
	Sender string `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
	// Message generation UNIX timestamp.
	Stamp int64 `protobuf:"varint,2,opt,name=stamp,proto3" json:"stamp,omitempty"`
}

func (x *GenericStreamChunk) Reset() {
	*x = GenericStreamChunk{}
	if protoimpl.UnsafeEnabled {
		mi := &file_sample_v1_foo_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GenericStreamChunk) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GenericStreamChunk) ProtoMessage() {}

func (x *GenericStreamChunk) ProtoReflect() protoreflect.Message {
	mi := &file_sample_v1_foo_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GenericStreamChunk.ProtoReflect.Descriptor instead.
func (*GenericStreamChunk) Descriptor() ([]byte, []int) {
	return file_sample_v1_foo_api_proto_rawDescGZIP(), []int{0}
}

func (x *GenericStreamChunk) GetSender() string {
	if x != nil {
		return x.Sender
	}
	return ""
}

func (x *GenericStreamChunk) GetStamp() int64 {
	if x != nil {
		return x.Stamp
	}
	return 0
}

// Generic stream messages send by the client.
type OpenClientStreamRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Identifier for the endpoint generating the stream message.
	Sender string `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
	// Message generation UNIX timestamp.
	Stamp int64 `protobuf:"varint,2,opt,name=stamp,proto3" json:"stamp,omitempty"`
}

func (x *OpenClientStreamRequest) Reset() {
	*x = OpenClientStreamRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_sample_v1_foo_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OpenClientStreamRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OpenClientStreamRequest) ProtoMessage() {}

func (x *OpenClientStreamRequest) ProtoReflect() protoreflect.Message {
	mi := &file_sample_v1_foo_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OpenClientStreamRequest.ProtoReflect.Descriptor instead.
func (*OpenClientStreamRequest) Descriptor() ([]byte, []int) {
	return file_sample_v1_foo_api_proto_rawDescGZIP(), []int{1}
}

func (x *OpenClientStreamRequest) GetSender() string {
	if x != nil {
		return x.Sender
	}
	return ""
}

func (x *OpenClientStreamRequest) GetStamp() int64 {
	if x != nil {
		return x.Stamp
	}
	return 0
}

// Generic stream result.
type StreamResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Set to the total number of messages received during the request.
	Received int64 `protobuf:"varint,1,opt,name=received,proto3" json:"received,omitempty"`
}

func (x *StreamResult) Reset() {
	*x = StreamResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_sample_v1_foo_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StreamResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StreamResult) ProtoMessage() {}

func (x *StreamResult) ProtoReflect() protoreflect.Message {
	mi := &file_sample_v1_foo_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StreamResult.ProtoReflect.Descriptor instead.
func (*StreamResult) Descriptor() ([]byte, []int) {
	return file_sample_v1_foo_api_proto_rawDescGZIP(), []int{2}
}

func (x *StreamResult) GetReceived() int64 {
	if x != nil {
		return x.Received
	}
	return 0
}

var File_sample_v1_foo_api_proto protoreflect.FileDescriptor

var file_sample_v1_foo_api_proto_rawDesc = []byte{
	0x0a, 0x17, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x66, 0x6f, 0x6f, 0x5f,
	0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x73, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x2e, 0x76, 0x31, 0x1a, 0x15, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2f, 0x76, 0x31, 0x2f,
	0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70,
	0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67,
	0x65, 0x6e, 0x2d, 0x6f, 0x70, 0x65, 0x6e, 0x61, 0x70, 0x69, 0x76, 0x32, 0x2f, 0x6f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x42, 0x0a, 0x12, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x69,
	0x63, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x12, 0x16, 0x0a, 0x06,
	0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x65,
	0x6e, 0x64, 0x65, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x05, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x22, 0x47, 0x0a, 0x17, 0x4f, 0x70,
	0x65, 0x6e, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x12, 0x14, 0x0a,
	0x05, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x22, 0x2a, 0x0a, 0x0c, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x65, 0x73,
	0x75, 0x6c, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x64, 0x32,
	0xe2, 0x04, 0x0a, 0x06, 0x46, 0x6f, 0x6f, 0x41, 0x50, 0x49, 0x12, 0x42, 0x0a, 0x04, 0x50, 0x69,
	0x6e, 0x67, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x0f, 0x2e, 0x73, 0x61, 0x6d,
	0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x6f, 0x6e, 0x67, 0x22, 0x11, 0x82, 0xd3, 0xe4,
	0x93, 0x02, 0x0b, 0x22, 0x09, 0x2f, 0x66, 0x6f, 0x6f, 0x2f, 0x70, 0x69, 0x6e, 0x67, 0x12, 0x50,
	0x0a, 0x06, 0x48, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x1a, 0x19, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61,
	0x6c, 0x74, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x13, 0x82, 0xd3, 0xe4,
	0x93, 0x02, 0x0d, 0x22, 0x0b, 0x2f, 0x66, 0x6f, 0x6f, 0x2f, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68,
	0x12, 0x4c, 0x0a, 0x07, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x16, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d,
	0x70, 0x74, 0x79, 0x1a, 0x13, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x14, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x0e,
	0x22, 0x0c, 0x2f, 0x66, 0x6f, 0x6f, 0x2f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x4f,
	0x0a, 0x06, 0x46, 0x61, 0x75, 0x6c, 0x74, 0x79, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x1a, 0x18, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x75, 0x6d,
	0x6d, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x13, 0x82, 0xd3, 0xe4, 0x93,
	0x02, 0x0d, 0x22, 0x0b, 0x2f, 0x66, 0x6f, 0x6f, 0x2f, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x79, 0x12,
	0x4b, 0x0a, 0x04, 0x53, 0x6c, 0x6f, 0x77, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a,
	0x18, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x75, 0x6d, 0x6d,
	0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x11, 0x82, 0xd3, 0xe4, 0x93, 0x02,
	0x0b, 0x22, 0x09, 0x2f, 0x66, 0x6f, 0x6f, 0x2f, 0x73, 0x6c, 0x6f, 0x77, 0x12, 0x67, 0x0a, 0x10,
	0x4f, 0x70, 0x65, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x1d, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x69, 0x63, 0x53, 0x74, 0x72, 0x65,
	0x61, 0x6d, 0x43, 0x68, 0x75, 0x6e, 0x6b, 0x22, 0x1a, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x14, 0x12,
	0x12, 0x2f, 0x66, 0x6f, 0x6f, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x72,
	0x65, 0x61, 0x6d, 0x30, 0x01, 0x12, 0x6d, 0x0a, 0x10, 0x4f, 0x70, 0x65, 0x6e, 0x43, 0x6c, 0x69,
	0x65, 0x6e, 0x74, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x22, 0x2e, 0x73, 0x61, 0x6d, 0x70,
	0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x4f, 0x70, 0x65, 0x6e, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e,
	0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d,
	0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x22, 0x1a, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x14, 0x12, 0x12,
	0x2f, 0x66, 0x6f, 0x6f, 0x2f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x73, 0x74, 0x72, 0x65,
	0x61, 0x6d, 0x28, 0x01, 0x42, 0xfc, 0x02, 0x92, 0x41, 0xf1, 0x01, 0x0a, 0x03, 0x32, 0x2e, 0x30,
	0x12, 0x32, 0x0a, 0x07, 0x46, 0x6f, 0x6f, 0x20, 0x41, 0x50, 0x49, 0x22, 0x20, 0x0a, 0x08, 0x4a,
	0x6f, 0x68, 0x6e, 0x20, 0x44, 0x6f, 0x65, 0x1a, 0x14, 0x6a, 0x6f, 0x68, 0x6e, 0x2e, 0x64, 0x6f,
	0x77, 0x40, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x63, 0x6f, 0x6d, 0x32, 0x05, 0x30,
	0x2e, 0x31, 0x2e, 0x30, 0x2a, 0x03, 0x01, 0x02, 0x04, 0x32, 0x10, 0x61, 0x70, 0x70, 0x6c, 0x69,
	0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x6a, 0x73, 0x6f, 0x6e, 0x32, 0x14, 0x61, 0x70, 0x70,
	0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x3a, 0x10, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x6a,
	0x73, 0x6f, 0x6e, 0x3a, 0x14, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x5a, 0x53, 0x0a, 0x51, 0x0a, 0x06, 0x62,
	0x65, 0x61, 0x72, 0x65, 0x72, 0x12, 0x47, 0x08, 0x02, 0x12, 0x32, 0x41, 0x75, 0x74, 0x68, 0x65,
	0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x20, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x20,
	0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x64, 0x20, 0x61, 0x73, 0x3a, 0x20, 0x27, 0x42, 0x65,
	0x61, 0x72, 0x65, 0x72, 0x20, 0x7b, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x7d, 0x27, 0x1a, 0x0d, 0x41,
	0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x20, 0x02, 0x62, 0x0c,
	0x0a, 0x0a, 0x0a, 0x06, 0x62, 0x65, 0x61, 0x72, 0x65, 0x72, 0x12, 0x00, 0x0a, 0x0d, 0x63, 0x6f,
	0x6d, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x42, 0x0b, 0x46, 0x6f, 0x6f,
	0x41, 0x70, 0x69, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x21, 0x67, 0x6f, 0x2e, 0x62,
	0x72, 0x79, 0x6b, 0x2e, 0x69, 0x6f, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x73, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x2f, 0x76, 0x31, 0x3b, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x76, 0x31, 0xf8, 0x01, 0x00,
	0xa2, 0x02, 0x03, 0x53, 0x58, 0x58, 0xaa, 0x02, 0x09, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e,
	0x56, 0x31, 0xca, 0x02, 0x09, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x5c, 0x56, 0x31, 0xe2, 0x02,
	0x15, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0a, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x3a,
	0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_sample_v1_foo_api_proto_rawDescOnce sync.Once
	file_sample_v1_foo_api_proto_rawDescData = file_sample_v1_foo_api_proto_rawDesc
)

func file_sample_v1_foo_api_proto_rawDescGZIP() []byte {
	file_sample_v1_foo_api_proto_rawDescOnce.Do(func() {
		file_sample_v1_foo_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_sample_v1_foo_api_proto_rawDescData)
	})
	return file_sample_v1_foo_api_proto_rawDescData
}

var file_sample_v1_foo_api_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_sample_v1_foo_api_proto_goTypes = []interface{}{
	(*GenericStreamChunk)(nil),      // 0: sample.v1.GenericStreamChunk
	(*OpenClientStreamRequest)(nil), // 1: sample.v1.OpenClientStreamRequest
	(*StreamResult)(nil),            // 2: sample.v1.StreamResult
	(*emptypb.Empty)(nil),           // 3: google.protobuf.Empty
	(*Pong)(nil),                    // 4: sample.v1.Pong
	(*HealthResponse)(nil),          // 5: sample.v1.HealthResponse
	(*Response)(nil),                // 6: sample.v1.Response
	(*DummyResponse)(nil),           // 7: sample.v1.DummyResponse
}
var file_sample_v1_foo_api_proto_depIdxs = []int32{
	3, // 0: sample.v1.FooAPI.Ping:input_type -> google.protobuf.Empty
	3, // 1: sample.v1.FooAPI.Health:input_type -> google.protobuf.Empty
	3, // 2: sample.v1.FooAPI.Request:input_type -> google.protobuf.Empty
	3, // 3: sample.v1.FooAPI.Faulty:input_type -> google.protobuf.Empty
	3, // 4: sample.v1.FooAPI.Slow:input_type -> google.protobuf.Empty
	3, // 5: sample.v1.FooAPI.OpenServerStream:input_type -> google.protobuf.Empty
	1, // 6: sample.v1.FooAPI.OpenClientStream:input_type -> sample.v1.OpenClientStreamRequest
	4, // 7: sample.v1.FooAPI.Ping:output_type -> sample.v1.Pong
	5, // 8: sample.v1.FooAPI.Health:output_type -> sample.v1.HealthResponse
	6, // 9: sample.v1.FooAPI.Request:output_type -> sample.v1.Response
	7, // 10: sample.v1.FooAPI.Faulty:output_type -> sample.v1.DummyResponse
	7, // 11: sample.v1.FooAPI.Slow:output_type -> sample.v1.DummyResponse
	0, // 12: sample.v1.FooAPI.OpenServerStream:output_type -> sample.v1.GenericStreamChunk
	2, // 13: sample.v1.FooAPI.OpenClientStream:output_type -> sample.v1.StreamResult
	7, // [7:14] is the sub-list for method output_type
	0, // [0:7] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_sample_v1_foo_api_proto_init() }
func file_sample_v1_foo_api_proto_init() {
	if File_sample_v1_foo_api_proto != nil {
		return
	}
	file_sample_v1_model_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_sample_v1_foo_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GenericStreamChunk); i {
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
		file_sample_v1_foo_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OpenClientStreamRequest); i {
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
		file_sample_v1_foo_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StreamResult); i {
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
			RawDescriptor: file_sample_v1_foo_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_sample_v1_foo_api_proto_goTypes,
		DependencyIndexes: file_sample_v1_foo_api_proto_depIdxs,
		MessageInfos:      file_sample_v1_foo_api_proto_msgTypes,
	}.Build()
	File_sample_v1_foo_api_proto = out.File
	file_sample_v1_foo_api_proto_rawDesc = nil
	file_sample_v1_foo_api_proto_goTypes = nil
	file_sample_v1_foo_api_proto_depIdxs = nil
}
