// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: test/v1/foo.proto

package testpb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	FooService_GetFoo_FullMethodName        = "/test.v1.FooService/GetFoo"
	FooService_ListFoos_FullMethodName      = "/test.v1.FooService/ListFoos"
	FooService_ListFooEvents_FullMethodName = "/test.v1.FooService/ListFooEvents"
)

// FooServiceClient is the client API for FooService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FooServiceClient interface {
	GetFoo(ctx context.Context, in *GetFooRequest, opts ...grpc.CallOption) (*GetFooResponse, error)
	ListFoos(ctx context.Context, in *ListFoosRequest, opts ...grpc.CallOption) (*ListFoosResponse, error)
	ListFooEvents(ctx context.Context, in *ListFooEventsRequest, opts ...grpc.CallOption) (*ListFooEventsResponse, error)
}

type fooServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewFooServiceClient(cc grpc.ClientConnInterface) FooServiceClient {
	return &fooServiceClient{cc}
}

func (c *fooServiceClient) GetFoo(ctx context.Context, in *GetFooRequest, opts ...grpc.CallOption) (*GetFooResponse, error) {
	out := new(GetFooResponse)
	err := c.cc.Invoke(ctx, FooService_GetFoo_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fooServiceClient) ListFoos(ctx context.Context, in *ListFoosRequest, opts ...grpc.CallOption) (*ListFoosResponse, error) {
	out := new(ListFoosResponse)
	err := c.cc.Invoke(ctx, FooService_ListFoos_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fooServiceClient) ListFooEvents(ctx context.Context, in *ListFooEventsRequest, opts ...grpc.CallOption) (*ListFooEventsResponse, error) {
	out := new(ListFooEventsResponse)
	err := c.cc.Invoke(ctx, FooService_ListFooEvents_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FooServiceServer is the server API for FooService service.
// All implementations must embed UnimplementedFooServiceServer
// for forward compatibility
type FooServiceServer interface {
	GetFoo(context.Context, *GetFooRequest) (*GetFooResponse, error)
	ListFoos(context.Context, *ListFoosRequest) (*ListFoosResponse, error)
	ListFooEvents(context.Context, *ListFooEventsRequest) (*ListFooEventsResponse, error)
	mustEmbedUnimplementedFooServiceServer()
}

// UnimplementedFooServiceServer must be embedded to have forward compatible implementations.
type UnimplementedFooServiceServer struct {
}

func (UnimplementedFooServiceServer) GetFoo(context.Context, *GetFooRequest) (*GetFooResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetFoo not implemented")
}
func (UnimplementedFooServiceServer) ListFoos(context.Context, *ListFoosRequest) (*ListFoosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListFoos not implemented")
}
func (UnimplementedFooServiceServer) ListFooEvents(context.Context, *ListFooEventsRequest) (*ListFooEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListFooEvents not implemented")
}
func (UnimplementedFooServiceServer) mustEmbedUnimplementedFooServiceServer() {}

// UnsafeFooServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FooServiceServer will
// result in compilation errors.
type UnsafeFooServiceServer interface {
	mustEmbedUnimplementedFooServiceServer()
}

func RegisterFooServiceServer(s grpc.ServiceRegistrar, srv FooServiceServer) {
	s.RegisterService(&FooService_ServiceDesc, srv)
}

func _FooService_GetFoo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetFooRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FooServiceServer).GetFoo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FooService_GetFoo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FooServiceServer).GetFoo(ctx, req.(*GetFooRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FooService_ListFoos_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListFoosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FooServiceServer).ListFoos(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FooService_ListFoos_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FooServiceServer).ListFoos(ctx, req.(*ListFoosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FooService_ListFooEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListFooEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FooServiceServer).ListFooEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FooService_ListFooEvents_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FooServiceServer).ListFooEvents(ctx, req.(*ListFooEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// FooService_ServiceDesc is the grpc.ServiceDesc for FooService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FooService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.v1.FooService",
	HandlerType: (*FooServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetFoo",
			Handler:    _FooService_GetFoo_Handler,
		},
		{
			MethodName: "ListFoos",
			Handler:    _FooService_ListFoos_Handler,
		},
		{
			MethodName: "ListFooEvents",
			Handler:    _FooService_ListFooEvents_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "test/v1/foo.proto",
}