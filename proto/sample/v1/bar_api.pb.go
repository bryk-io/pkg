// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        buf-v1.11.0
// source: sample/v1/bar_api.proto

package samplev1

import (
	reflect "reflect"

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

var File_sample_v1_bar_api_proto protoreflect.FileDescriptor

var file_sample_v1_bar_api_proto_rawDesc = []byte{
	0x0a, 0x17, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x62, 0x61, 0x72, 0x5f,
	0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x73, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x2e, 0x76, 0x31, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x15, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x6f, 0x64,
	0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x28, 0x74, 0x68, 0x69, 0x72, 0x64, 0x5f,
	0x70, 0x61, 0x72, 0x74, 0x79, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x27, 0x74, 0x68, 0x69, 0x72, 0x64, 0x5f, 0x70, 0x61, 0x72, 0x74, 0x79, 0x2f,
	0x6f, 0x70, 0x65, 0x6e, 0x61, 0x70, 0x69, 0x76, 0x32, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x32, 0xec, 0x01, 0x0a, 0x06,
	0x42, 0x61, 0x72, 0x41, 0x50, 0x49, 0x12, 0x42, 0x0a, 0x04, 0x50, 0x69, 0x6e, 0x67, 0x12, 0x16,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x0f, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e,
	0x76, 0x31, 0x2e, 0x50, 0x6f, 0x6e, 0x67, 0x22, 0x11, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x0b, 0x22,
	0x09, 0x2f, 0x62, 0x61, 0x72, 0x2f, 0x70, 0x69, 0x6e, 0x67, 0x12, 0x50, 0x0a, 0x06, 0x48, 0x65,
	0x61, 0x6c, 0x74, 0x68, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x19, 0x2e, 0x73,
	0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x13, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x0d, 0x22,
	0x0b, 0x2f, 0x62, 0x61, 0x72, 0x2f, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x12, 0x4c, 0x0a, 0x07,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a,
	0x13, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x14, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x0e, 0x22, 0x0c, 0x2f, 0x62,
	0x61, 0x72, 0x2f, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x42, 0x85, 0x02, 0x0a, 0x0d, 0x63,
	0x6f, 0x6d, 0x2e, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x42, 0x0b, 0x42, 0x61,
	0x72, 0x41, 0x70, 0x69, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x12, 0x73, 0x61, 0x6d,
	0x70, 0x6c, 0x65, 0x2f, 0x76, 0x31, 0x3b, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x76, 0x31, 0xf8,
	0x01, 0x00, 0xa2, 0x02, 0x03, 0x53, 0x58, 0x58, 0xaa, 0x02, 0x09, 0x53, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x09, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x5c, 0x56, 0x31,
	0xe2, 0x02, 0x15, 0x53, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42,
	0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0a, 0x53, 0x61, 0x6d, 0x70, 0x6c,
	0x65, 0x3a, 0x3a, 0x56, 0x31, 0x92, 0x41, 0x89, 0x01, 0x12, 0x32, 0x0a, 0x07, 0x42, 0x61, 0x72,
	0x20, 0x41, 0x50, 0x49, 0x22, 0x20, 0x0a, 0x08, 0x4a, 0x6f, 0x68, 0x6e, 0x20, 0x44, 0x6f, 0x65,
	0x1a, 0x14, 0x6a, 0x6f, 0x68, 0x6e, 0x2e, 0x64, 0x6f, 0x77, 0x40, 0x65, 0x78, 0x61, 0x6d, 0x70,
	0x6c, 0x65, 0x2e, 0x63, 0x6f, 0x6d, 0x32, 0x05, 0x30, 0x2e, 0x31, 0x2e, 0x30, 0x2a, 0x03, 0x01,
	0x02, 0x04, 0x32, 0x10, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f,
	0x6a, 0x73, 0x6f, 0x6e, 0x32, 0x14, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x3a, 0x10, 0x61, 0x70, 0x70, 0x6c,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x6a, 0x73, 0x6f, 0x6e, 0x3a, 0x14, 0x61, 0x70,
	0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_sample_v1_bar_api_proto_goTypes = []interface{}{
	(*emptypb.Empty)(nil),  // 0: google.protobuf.Empty
	(*Pong)(nil),           // 1: sample.v1.Pong
	(*HealthResponse)(nil), // 2: sample.v1.HealthResponse
	(*Response)(nil),       // 3: sample.v1.Response
}
var file_sample_v1_bar_api_proto_depIdxs = []int32{
	0, // 0: sample.v1.BarAPI.Ping:input_type -> google.protobuf.Empty
	0, // 1: sample.v1.BarAPI.Health:input_type -> google.protobuf.Empty
	0, // 2: sample.v1.BarAPI.Request:input_type -> google.protobuf.Empty
	1, // 3: sample.v1.BarAPI.Ping:output_type -> sample.v1.Pong
	2, // 4: sample.v1.BarAPI.Health:output_type -> sample.v1.HealthResponse
	3, // 5: sample.v1.BarAPI.Request:output_type -> sample.v1.Response
	3, // [3:6] is the sub-list for method output_type
	0, // [0:3] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_sample_v1_bar_api_proto_init() }
func file_sample_v1_bar_api_proto_init() {
	if File_sample_v1_bar_api_proto != nil {
		return
	}
	file_sample_v1_model_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_sample_v1_bar_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_sample_v1_bar_api_proto_goTypes,
		DependencyIndexes: file_sample_v1_bar_api_proto_depIdxs,
	}.Build()
	File_sample_v1_bar_api_proto = out.File
	file_sample_v1_bar_api_proto_rawDesc = nil
	file_sample_v1_bar_api_proto_goTypes = nil
	file_sample_v1_bar_api_proto_depIdxs = nil
}
