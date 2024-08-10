// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             (unknown)
// source: test/v1/service/bar.proto

package test_spb

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.62.0 or later.
const _ = grpc.SupportPackageIsVersion8

const (
	BarService_GetBar_FullMethodName        = "/test.v1.service.BarService/GetBar"
	BarService_ListBars_FullMethodName      = "/test.v1.service.BarService/ListBars"
	BarService_ListBarEvents_FullMethodName = "/test.v1.service.BarService/ListBarEvents"
)

// BarServiceClient is the client API for BarService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// BarService is a little strange, it requires a lot of overrides for defaults
// when building the state query and state machine.
type BarServiceClient interface {
	GetBar(ctx context.Context, in *GetBarRequest, opts ...grpc.CallOption) (*GetBarResponse, error)
	ListBars(ctx context.Context, in *ListBarsRequest, opts ...grpc.CallOption) (*ListBarsResponse, error)
	ListBarEvents(ctx context.Context, in *ListBarEventsRequest, opts ...grpc.CallOption) (*ListBarEventsResponse, error)
}

type barServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBarServiceClient(cc grpc.ClientConnInterface) BarServiceClient {
	return &barServiceClient{cc}
}

func (c *barServiceClient) GetBar(ctx context.Context, in *GetBarRequest, opts ...grpc.CallOption) (*GetBarResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetBarResponse)
	err := c.cc.Invoke(ctx, BarService_GetBar_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *barServiceClient) ListBars(ctx context.Context, in *ListBarsRequest, opts ...grpc.CallOption) (*ListBarsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListBarsResponse)
	err := c.cc.Invoke(ctx, BarService_ListBars_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *barServiceClient) ListBarEvents(ctx context.Context, in *ListBarEventsRequest, opts ...grpc.CallOption) (*ListBarEventsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListBarEventsResponse)
	err := c.cc.Invoke(ctx, BarService_ListBarEvents_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BarServiceServer is the server API for BarService service.
// All implementations must embed UnimplementedBarServiceServer
// for forward compatibility
//
// BarService is a little strange, it requires a lot of overrides for defaults
// when building the state query and state machine.
type BarServiceServer interface {
	GetBar(context.Context, *GetBarRequest) (*GetBarResponse, error)
	ListBars(context.Context, *ListBarsRequest) (*ListBarsResponse, error)
	ListBarEvents(context.Context, *ListBarEventsRequest) (*ListBarEventsResponse, error)
	mustEmbedUnimplementedBarServiceServer()
}

// UnimplementedBarServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBarServiceServer struct {
}

func (UnimplementedBarServiceServer) GetBar(context.Context, *GetBarRequest) (*GetBarResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBar not implemented")
}
func (UnimplementedBarServiceServer) ListBars(context.Context, *ListBarsRequest) (*ListBarsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListBars not implemented")
}
func (UnimplementedBarServiceServer) ListBarEvents(context.Context, *ListBarEventsRequest) (*ListBarEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListBarEvents not implemented")
}
func (UnimplementedBarServiceServer) mustEmbedUnimplementedBarServiceServer() {}

// UnsafeBarServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BarServiceServer will
// result in compilation errors.
type UnsafeBarServiceServer interface {
	mustEmbedUnimplementedBarServiceServer()
}

func RegisterBarServiceServer(s grpc.ServiceRegistrar, srv BarServiceServer) {
	s.RegisterService(&BarService_ServiceDesc, srv)
}

func _BarService_GetBar_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBarRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BarServiceServer).GetBar(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BarService_GetBar_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BarServiceServer).GetBar(ctx, req.(*GetBarRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BarService_ListBars_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListBarsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BarServiceServer).ListBars(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BarService_ListBars_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BarServiceServer).ListBars(ctx, req.(*ListBarsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BarService_ListBarEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListBarEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BarServiceServer).ListBarEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BarService_ListBarEvents_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BarServiceServer).ListBarEvents(ctx, req.(*ListBarEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BarService_ServiceDesc is the grpc.ServiceDesc for BarService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BarService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.v1.service.BarService",
	HandlerType: (*BarServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetBar",
			Handler:    _BarService_GetBar_Handler,
		},
		{
			MethodName: "ListBars",
			Handler:    _BarService_ListBars_Handler,
		},
		{
			MethodName: "ListBarEvents",
			Handler:    _BarService_ListBarEvents_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "test/v1/service/bar.proto",
}
