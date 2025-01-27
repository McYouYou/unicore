// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v5.27.2
// source: biz.proto

package biz

import (
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

type Service struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      string         `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	OwnerId string         `protobuf:"bytes,2,opt,name=ownerId,proto3" json:"ownerId,omitempty"`
	Admins  []string       `protobuf:"bytes,3,rep,name=admins,proto3" json:"admins,omitempty"`
	Name    string         `protobuf:"bytes,4,opt,name=name,proto3" json:"name,omitempty"`
	Option  *ServiceOption `protobuf:"bytes,5,opt,name=option,proto3" json:"option,omitempty"`
}

func (x *Service) Reset() {
	*x = Service{}
	if protoimpl.UnsafeEnabled {
		mi := &file_biz_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Service) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Service) ProtoMessage() {}

func (x *Service) ProtoReflect() protoreflect.Message {
	mi := &file_biz_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Service.ProtoReflect.Descriptor instead.
func (*Service) Descriptor() ([]byte, []int) {
	return file_biz_proto_rawDescGZIP(), []int{0}
}

func (x *Service) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Service) GetOwnerId() string {
	if x != nil {
		return x.OwnerId
	}
	return ""
}

func (x *Service) GetAdmins() []string {
	if x != nil {
		return x.Admins
	}
	return nil
}

func (x *Service) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Service) GetOption() *ServiceOption {
	if x != nil {
		return x.Option
	}
	return nil
}

type ServiceOption struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Stateful       bool            `protobuf:"varint,1,opt,name=stateful,proto3" json:"stateful,omitempty"`
	Replicas       int32           `protobuf:"varint,2,opt,name=replicas,proto3" json:"replicas,omitempty"`
	WithStagingEnv bool            `protobuf:"varint,3,opt,name=withStagingEnv,proto3" json:"withStagingEnv,omitempty"`
	Resource       *ResourceOption `protobuf:"bytes,4,opt,name=resource,proto3" json:"resource,omitempty"`
}

func (x *ServiceOption) Reset() {
	*x = ServiceOption{}
	if protoimpl.UnsafeEnabled {
		mi := &file_biz_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ServiceOption) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServiceOption) ProtoMessage() {}

func (x *ServiceOption) ProtoReflect() protoreflect.Message {
	mi := &file_biz_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServiceOption.ProtoReflect.Descriptor instead.
func (*ServiceOption) Descriptor() ([]byte, []int) {
	return file_biz_proto_rawDescGZIP(), []int{1}
}

func (x *ServiceOption) GetStateful() bool {
	if x != nil {
		return x.Stateful
	}
	return false
}

func (x *ServiceOption) GetReplicas() int32 {
	if x != nil {
		return x.Replicas
	}
	return 0
}

func (x *ServiceOption) GetWithStagingEnv() bool {
	if x != nil {
		return x.WithStagingEnv
	}
	return false
}

func (x *ServiceOption) GetResource() *ResourceOption {
	if x != nil {
		return x.Resource
	}
	return nil
}

type ResourceOption struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Cpus         int32 `protobuf:"varint,1,opt,name=cpus,proto3" json:"cpus,omitempty"`
	MemInMb      int32 `protobuf:"varint,2,opt,name=memInMb,proto3" json:"memInMb,omitempty"`
	StorageMb    int32 `protobuf:"varint,3,opt,name=storageMb,proto3" json:"storageMb,omitempty"`
	FilesystemMb int32 `protobuf:"varint,4,opt,name=filesystemMb,proto3" json:"filesystemMb,omitempty"`
}

func (x *ResourceOption) Reset() {
	*x = ResourceOption{}
	if protoimpl.UnsafeEnabled {
		mi := &file_biz_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResourceOption) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceOption) ProtoMessage() {}

func (x *ResourceOption) ProtoReflect() protoreflect.Message {
	mi := &file_biz_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceOption.ProtoReflect.Descriptor instead.
func (*ResourceOption) Descriptor() ([]byte, []int) {
	return file_biz_proto_rawDescGZIP(), []int{2}
}

func (x *ResourceOption) GetCpus() int32 {
	if x != nil {
		return x.Cpus
	}
	return 0
}

func (x *ResourceOption) GetMemInMb() int32 {
	if x != nil {
		return x.MemInMb
	}
	return 0
}

func (x *ResourceOption) GetStorageMb() int32 {
	if x != nil {
		return x.StorageMb
	}
	return 0
}

func (x *ResourceOption) GetFilesystemMb() int32 {
	if x != nil {
		return x.FilesystemMb
	}
	return 0
}

var File_biz_proto protoreflect.FileDescriptor

var file_biz_proto_rawDesc = []byte{
	0x0a, 0x09, 0x62, 0x69, 0x7a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x03, 0x62, 0x69, 0x7a,
	0x22, 0x8b, 0x01, 0x0a, 0x07, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07,
	0x6f, 0x77, 0x6e, 0x65, 0x72, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6f,
	0x77, 0x6e, 0x65, 0x72, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x73,
	0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x61, 0x64, 0x6d, 0x69, 0x6e, 0x73, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x12, 0x2a, 0x0a, 0x06, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x12, 0x2e, 0x62, 0x69, 0x7a, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x06, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0xa0,
	0x01, 0x0a, 0x0d, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x1a, 0x0a, 0x08, 0x73, 0x74, 0x61, 0x74, 0x65, 0x66, 0x75, 0x6c, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x08, 0x73, 0x74, 0x61, 0x74, 0x65, 0x66, 0x75, 0x6c, 0x12, 0x1a, 0x0a, 0x08,
	0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08,
	0x72, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x73, 0x12, 0x26, 0x0a, 0x0e, 0x77, 0x69, 0x74, 0x68,
	0x53, 0x74, 0x61, 0x67, 0x69, 0x6e, 0x67, 0x45, 0x6e, 0x76, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x0e, 0x77, 0x69, 0x74, 0x68, 0x53, 0x74, 0x61, 0x67, 0x69, 0x6e, 0x67, 0x45, 0x6e, 0x76,
	0x12, 0x2f, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x13, 0x2e, 0x62, 0x69, 0x7a, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x22, 0x80, 0x01, 0x0a, 0x0e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x4f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x70, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x04, 0x63, 0x70, 0x75, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x6d, 0x49,
	0x6e, 0x4d, 0x62, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x6d, 0x65, 0x6d, 0x49, 0x6e,
	0x4d, 0x62, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x4d, 0x62, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x4d, 0x62,
	0x12, 0x22, 0x0a, 0x0c, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x4d, 0x62,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0c, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x79, 0x73, 0x74,
	0x65, 0x6d, 0x4d, 0x62, 0x42, 0x13, 0x5a, 0x11, 0x75, 0x6e, 0x69, 0x63, 0x6f, 0x72, 0x65, 0x2f,
	0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2f, 0x62, 0x69, 0x7a, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_biz_proto_rawDescOnce sync.Once
	file_biz_proto_rawDescData = file_biz_proto_rawDesc
)

func file_biz_proto_rawDescGZIP() []byte {
	file_biz_proto_rawDescOnce.Do(func() {
		file_biz_proto_rawDescData = protoimpl.X.CompressGZIP(file_biz_proto_rawDescData)
	})
	return file_biz_proto_rawDescData
}

var file_biz_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_biz_proto_goTypes = []any{
	(*Service)(nil),        // 0: biz.Service
	(*ServiceOption)(nil),  // 1: biz.ServiceOption
	(*ResourceOption)(nil), // 2: biz.ResourceOption
}
var file_biz_proto_depIdxs = []int32{
	1, // 0: biz.Service.option:type_name -> biz.ServiceOption
	2, // 1: biz.ServiceOption.resource:type_name -> biz.ResourceOption
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_biz_proto_init() }
func file_biz_proto_init() {
	if File_biz_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_biz_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*Service); i {
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
		file_biz_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*ServiceOption); i {
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
		file_biz_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*ResourceOption); i {
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
			RawDescriptor: file_biz_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_biz_proto_goTypes,
		DependencyIndexes: file_biz_proto_depIdxs,
		MessageInfos:      file_biz_proto_msgTypes,
	}.Build()
	File_biz_proto = out.File
	file_biz_proto_rawDesc = nil
	file_biz_proto_goTypes = nil
	file_biz_proto_depIdxs = nil
}
