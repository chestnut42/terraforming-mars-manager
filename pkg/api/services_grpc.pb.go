// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.4.0
// - protoc             v5.27.2
// source: pkg/api/services.proto

package api

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
	Users_Login_FullMethodName             = "/api.Users/Login"
	Users_GetMe_FullMethodName             = "/api.Users/GetMe"
	Users_UpdateMe_FullMethodName          = "/api.Users/UpdateMe"
	Users_UpdateDeviceToken_FullMethodName = "/api.Users/UpdateDeviceToken"
	Users_SearchUser_FullMethodName        = "/api.Users/SearchUser"
)

// UsersClient is the client API for Users service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type UsersClient interface {
	Login(ctx context.Context, in *Login_Request, opts ...grpc.CallOption) (*Login_Response, error)
	GetMe(ctx context.Context, in *GetMe_Request, opts ...grpc.CallOption) (*GetMe_Response, error)
	UpdateMe(ctx context.Context, in *UpdateMe_Request, opts ...grpc.CallOption) (*UpdateMe_Response, error)
	UpdateDeviceToken(ctx context.Context, in *UpdateDeviceToken_Request, opts ...grpc.CallOption) (*UpdateDeviceToken_Response, error)
	SearchUser(ctx context.Context, in *SearchUser_Request, opts ...grpc.CallOption) (*SearchUser_Response, error)
}

type usersClient struct {
	cc grpc.ClientConnInterface
}

func NewUsersClient(cc grpc.ClientConnInterface) UsersClient {
	return &usersClient{cc}
}

func (c *usersClient) Login(ctx context.Context, in *Login_Request, opts ...grpc.CallOption) (*Login_Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Login_Response)
	err := c.cc.Invoke(ctx, Users_Login_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *usersClient) GetMe(ctx context.Context, in *GetMe_Request, opts ...grpc.CallOption) (*GetMe_Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetMe_Response)
	err := c.cc.Invoke(ctx, Users_GetMe_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *usersClient) UpdateMe(ctx context.Context, in *UpdateMe_Request, opts ...grpc.CallOption) (*UpdateMe_Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdateMe_Response)
	err := c.cc.Invoke(ctx, Users_UpdateMe_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *usersClient) UpdateDeviceToken(ctx context.Context, in *UpdateDeviceToken_Request, opts ...grpc.CallOption) (*UpdateDeviceToken_Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdateDeviceToken_Response)
	err := c.cc.Invoke(ctx, Users_UpdateDeviceToken_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *usersClient) SearchUser(ctx context.Context, in *SearchUser_Request, opts ...grpc.CallOption) (*SearchUser_Response, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SearchUser_Response)
	err := c.cc.Invoke(ctx, Users_SearchUser_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UsersServer is the server API for Users service.
// All implementations must embed UnimplementedUsersServer
// for forward compatibility
type UsersServer interface {
	Login(context.Context, *Login_Request) (*Login_Response, error)
	GetMe(context.Context, *GetMe_Request) (*GetMe_Response, error)
	UpdateMe(context.Context, *UpdateMe_Request) (*UpdateMe_Response, error)
	UpdateDeviceToken(context.Context, *UpdateDeviceToken_Request) (*UpdateDeviceToken_Response, error)
	SearchUser(context.Context, *SearchUser_Request) (*SearchUser_Response, error)
	mustEmbedUnimplementedUsersServer()
}

// UnimplementedUsersServer must be embedded to have forward compatible implementations.
type UnimplementedUsersServer struct {
}

func (UnimplementedUsersServer) Login(context.Context, *Login_Request) (*Login_Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}
func (UnimplementedUsersServer) GetMe(context.Context, *GetMe_Request) (*GetMe_Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetMe not implemented")
}
func (UnimplementedUsersServer) UpdateMe(context.Context, *UpdateMe_Request) (*UpdateMe_Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateMe not implemented")
}
func (UnimplementedUsersServer) UpdateDeviceToken(context.Context, *UpdateDeviceToken_Request) (*UpdateDeviceToken_Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateDeviceToken not implemented")
}
func (UnimplementedUsersServer) SearchUser(context.Context, *SearchUser_Request) (*SearchUser_Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SearchUser not implemented")
}
func (UnimplementedUsersServer) mustEmbedUnimplementedUsersServer() {}

// UnsafeUsersServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to UsersServer will
// result in compilation errors.
type UnsafeUsersServer interface {
	mustEmbedUnimplementedUsersServer()
}

func RegisterUsersServer(s grpc.ServiceRegistrar, srv UsersServer) {
	s.RegisterService(&Users_ServiceDesc, srv)
}

func _Users_Login_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Login_Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UsersServer).Login(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Users_Login_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UsersServer).Login(ctx, req.(*Login_Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Users_GetMe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetMe_Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UsersServer).GetMe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Users_GetMe_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UsersServer).GetMe(ctx, req.(*GetMe_Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Users_UpdateMe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateMe_Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UsersServer).UpdateMe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Users_UpdateMe_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UsersServer).UpdateMe(ctx, req.(*UpdateMe_Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Users_UpdateDeviceToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateDeviceToken_Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UsersServer).UpdateDeviceToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Users_UpdateDeviceToken_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UsersServer).UpdateDeviceToken(ctx, req.(*UpdateDeviceToken_Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Users_SearchUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchUser_Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UsersServer).SearchUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Users_SearchUser_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UsersServer).SearchUser(ctx, req.(*SearchUser_Request))
	}
	return interceptor(ctx, in, info, handler)
}

// Users_ServiceDesc is the grpc.ServiceDesc for Users service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Users_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.Users",
	HandlerType: (*UsersServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Login",
			Handler:    _Users_Login_Handler,
		},
		{
			MethodName: "GetMe",
			Handler:    _Users_GetMe_Handler,
		},
		{
			MethodName: "UpdateMe",
			Handler:    _Users_UpdateMe_Handler,
		},
		{
			MethodName: "UpdateDeviceToken",
			Handler:    _Users_UpdateDeviceToken_Handler,
		},
		{
			MethodName: "SearchUser",
			Handler:    _Users_SearchUser_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pkg/api/services.proto",
}