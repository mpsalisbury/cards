// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: game.proto

package proto

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

// CardGameServiceClient is the client API for CardGameService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CardGameServiceClient interface {
	Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error)
	Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
	ListGames(ctx context.Context, in *ListGamesRequest, opts ...grpc.CallOption) (*ListGamesResponse, error)
	JoinGame(ctx context.Context, in *JoinGameRequest, opts ...grpc.CallOption) (*JoinGameResponse, error)
	GameAction(ctx context.Context, in *GameActionRequest, opts ...grpc.CallOption) (*Status, error)
	GetGameState(ctx context.Context, in *GameStateRequest, opts ...grpc.CallOption) (*GameState, error)
	ListenForGameActivity(ctx context.Context, in *GameActivityRequest, opts ...grpc.CallOption) (CardGameService_ListenForGameActivityClient, error)
}

type cardGameServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCardGameServiceClient(cc grpc.ClientConnInterface) CardGameServiceClient {
	return &cardGameServiceClient{cc}
}

func (c *cardGameServiceClient) Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error) {
	out := new(PingResponse)
	err := c.cc.Invoke(ctx, "/cards.proto.CardGameService/Ping", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cardGameServiceClient) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error) {
	out := new(RegisterResponse)
	err := c.cc.Invoke(ctx, "/cards.proto.CardGameService/Register", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cardGameServiceClient) ListGames(ctx context.Context, in *ListGamesRequest, opts ...grpc.CallOption) (*ListGamesResponse, error) {
	out := new(ListGamesResponse)
	err := c.cc.Invoke(ctx, "/cards.proto.CardGameService/ListGames", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cardGameServiceClient) JoinGame(ctx context.Context, in *JoinGameRequest, opts ...grpc.CallOption) (*JoinGameResponse, error) {
	out := new(JoinGameResponse)
	err := c.cc.Invoke(ctx, "/cards.proto.CardGameService/JoinGame", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cardGameServiceClient) GameAction(ctx context.Context, in *GameActionRequest, opts ...grpc.CallOption) (*Status, error) {
	out := new(Status)
	err := c.cc.Invoke(ctx, "/cards.proto.CardGameService/GameAction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cardGameServiceClient) GetGameState(ctx context.Context, in *GameStateRequest, opts ...grpc.CallOption) (*GameState, error) {
	out := new(GameState)
	err := c.cc.Invoke(ctx, "/cards.proto.CardGameService/GetGameState", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cardGameServiceClient) ListenForGameActivity(ctx context.Context, in *GameActivityRequest, opts ...grpc.CallOption) (CardGameService_ListenForGameActivityClient, error) {
	stream, err := c.cc.NewStream(ctx, &CardGameService_ServiceDesc.Streams[0], "/cards.proto.CardGameService/ListenForGameActivity", opts...)
	if err != nil {
		return nil, err
	}
	x := &cardGameServiceListenForGameActivityClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type CardGameService_ListenForGameActivityClient interface {
	Recv() (*GameActivityResponse, error)
	grpc.ClientStream
}

type cardGameServiceListenForGameActivityClient struct {
	grpc.ClientStream
}

func (x *cardGameServiceListenForGameActivityClient) Recv() (*GameActivityResponse, error) {
	m := new(GameActivityResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// CardGameServiceServer is the server API for CardGameService service.
// All implementations must embed UnimplementedCardGameServiceServer
// for forward compatibility
type CardGameServiceServer interface {
	Ping(context.Context, *PingRequest) (*PingResponse, error)
	Register(context.Context, *RegisterRequest) (*RegisterResponse, error)
	ListGames(context.Context, *ListGamesRequest) (*ListGamesResponse, error)
	JoinGame(context.Context, *JoinGameRequest) (*JoinGameResponse, error)
	GameAction(context.Context, *GameActionRequest) (*Status, error)
	GetGameState(context.Context, *GameStateRequest) (*GameState, error)
	ListenForGameActivity(*GameActivityRequest, CardGameService_ListenForGameActivityServer) error
	mustEmbedUnimplementedCardGameServiceServer()
}

// UnimplementedCardGameServiceServer must be embedded to have forward compatible implementations.
type UnimplementedCardGameServiceServer struct {
}

func (UnimplementedCardGameServiceServer) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedCardGameServiceServer) Register(context.Context, *RegisterRequest) (*RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (UnimplementedCardGameServiceServer) ListGames(context.Context, *ListGamesRequest) (*ListGamesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListGames not implemented")
}
func (UnimplementedCardGameServiceServer) JoinGame(context.Context, *JoinGameRequest) (*JoinGameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method JoinGame not implemented")
}
func (UnimplementedCardGameServiceServer) GameAction(context.Context, *GameActionRequest) (*Status, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GameAction not implemented")
}
func (UnimplementedCardGameServiceServer) GetGameState(context.Context, *GameStateRequest) (*GameState, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetGameState not implemented")
}
func (UnimplementedCardGameServiceServer) ListenForGameActivity(*GameActivityRequest, CardGameService_ListenForGameActivityServer) error {
	return status.Errorf(codes.Unimplemented, "method ListenForGameActivity not implemented")
}
func (UnimplementedCardGameServiceServer) mustEmbedUnimplementedCardGameServiceServer() {}

// UnsafeCardGameServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CardGameServiceServer will
// result in compilation errors.
type UnsafeCardGameServiceServer interface {
	mustEmbedUnimplementedCardGameServiceServer()
}

func RegisterCardGameServiceServer(s grpc.ServiceRegistrar, srv CardGameServiceServer) {
	s.RegisterService(&CardGameService_ServiceDesc, srv)
}

func _CardGameService_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardGameServiceServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cards.proto.CardGameService/Ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardGameServiceServer).Ping(ctx, req.(*PingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CardGameService_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardGameServiceServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cards.proto.CardGameService/Register",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardGameServiceServer).Register(ctx, req.(*RegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CardGameService_ListGames_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListGamesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardGameServiceServer).ListGames(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cards.proto.CardGameService/ListGames",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardGameServiceServer).ListGames(ctx, req.(*ListGamesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CardGameService_JoinGame_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(JoinGameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardGameServiceServer).JoinGame(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cards.proto.CardGameService/JoinGame",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardGameServiceServer).JoinGame(ctx, req.(*JoinGameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CardGameService_GameAction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GameActionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardGameServiceServer).GameAction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cards.proto.CardGameService/GameAction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardGameServiceServer).GameAction(ctx, req.(*GameActionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CardGameService_GetGameState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GameStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardGameServiceServer).GetGameState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cards.proto.CardGameService/GetGameState",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardGameServiceServer).GetGameState(ctx, req.(*GameStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CardGameService_ListenForGameActivity_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GameActivityRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CardGameServiceServer).ListenForGameActivity(m, &cardGameServiceListenForGameActivityServer{stream})
}

type CardGameService_ListenForGameActivityServer interface {
	Send(*GameActivityResponse) error
	grpc.ServerStream
}

type cardGameServiceListenForGameActivityServer struct {
	grpc.ServerStream
}

func (x *cardGameServiceListenForGameActivityServer) Send(m *GameActivityResponse) error {
	return x.ServerStream.SendMsg(m)
}

// CardGameService_ServiceDesc is the grpc.ServiceDesc for CardGameService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var CardGameService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cards.proto.CardGameService",
	HandlerType: (*CardGameServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _CardGameService_Ping_Handler,
		},
		{
			MethodName: "Register",
			Handler:    _CardGameService_Register_Handler,
		},
		{
			MethodName: "ListGames",
			Handler:    _CardGameService_ListGames_Handler,
		},
		{
			MethodName: "JoinGame",
			Handler:    _CardGameService_JoinGame_Handler,
		},
		{
			MethodName: "GameAction",
			Handler:    _CardGameService_GameAction_Handler,
		},
		{
			MethodName: "GetGameState",
			Handler:    _CardGameService_GetGameState_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ListenForGameActivity",
			Handler:       _CardGameService_ListenForGameActivity_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "game.proto",
}
