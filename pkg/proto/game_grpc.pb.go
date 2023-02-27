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
	Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (CardGameService_RegisterClient, error)
	CreateGame(ctx context.Context, in *CreateGameRequest, opts ...grpc.CallOption) (*CreateGameResponse, error)
	ListGames(ctx context.Context, in *ListGamesRequest, opts ...grpc.CallOption) (*ListGamesResponse, error)
	JoinGame(ctx context.Context, in *JoinGameRequest, opts ...grpc.CallOption) (CardGameService_JoinGameClient, error)
	ObserveGame(ctx context.Context, in *ObserveGameRequest, opts ...grpc.CallOption) (CardGameService_ObserveGameClient, error)
	GameAction(ctx context.Context, in *GameActionRequest, opts ...grpc.CallOption) (*Status, error)
	GetGameState(ctx context.Context, in *GameStateRequest, opts ...grpc.CallOption) (*GameState, error)
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

func (c *cardGameServiceClient) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (CardGameService_RegisterClient, error) {
	stream, err := c.cc.NewStream(ctx, &CardGameService_ServiceDesc.Streams[0], "/cards.proto.CardGameService/Register", opts...)
	if err != nil {
		return nil, err
	}
	x := &cardGameServiceRegisterClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type CardGameService_RegisterClient interface {
	Recv() (*RegistryActivity, error)
	grpc.ClientStream
}

type cardGameServiceRegisterClient struct {
	grpc.ClientStream
}

func (x *cardGameServiceRegisterClient) Recv() (*RegistryActivity, error) {
	m := new(RegistryActivity)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *cardGameServiceClient) CreateGame(ctx context.Context, in *CreateGameRequest, opts ...grpc.CallOption) (*CreateGameResponse, error) {
	out := new(CreateGameResponse)
	err := c.cc.Invoke(ctx, "/cards.proto.CardGameService/CreateGame", in, out, opts...)
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

func (c *cardGameServiceClient) JoinGame(ctx context.Context, in *JoinGameRequest, opts ...grpc.CallOption) (CardGameService_JoinGameClient, error) {
	stream, err := c.cc.NewStream(ctx, &CardGameService_ServiceDesc.Streams[1], "/cards.proto.CardGameService/JoinGame", opts...)
	if err != nil {
		return nil, err
	}
	x := &cardGameServiceJoinGameClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type CardGameService_JoinGameClient interface {
	Recv() (*GameActivity, error)
	grpc.ClientStream
}

type cardGameServiceJoinGameClient struct {
	grpc.ClientStream
}

func (x *cardGameServiceJoinGameClient) Recv() (*GameActivity, error) {
	m := new(GameActivity)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *cardGameServiceClient) ObserveGame(ctx context.Context, in *ObserveGameRequest, opts ...grpc.CallOption) (CardGameService_ObserveGameClient, error) {
	stream, err := c.cc.NewStream(ctx, &CardGameService_ServiceDesc.Streams[2], "/cards.proto.CardGameService/ObserveGame", opts...)
	if err != nil {
		return nil, err
	}
	x := &cardGameServiceObserveGameClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type CardGameService_ObserveGameClient interface {
	Recv() (*GameActivity, error)
	grpc.ClientStream
}

type cardGameServiceObserveGameClient struct {
	grpc.ClientStream
}

func (x *cardGameServiceObserveGameClient) Recv() (*GameActivity, error) {
	m := new(GameActivity)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
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

// CardGameServiceServer is the server API for CardGameService service.
// All implementations must embed UnimplementedCardGameServiceServer
// for forward compatibility
type CardGameServiceServer interface {
	Ping(context.Context, *PingRequest) (*PingResponse, error)
	Register(*RegisterRequest, CardGameService_RegisterServer) error
	CreateGame(context.Context, *CreateGameRequest) (*CreateGameResponse, error)
	ListGames(context.Context, *ListGamesRequest) (*ListGamesResponse, error)
	JoinGame(*JoinGameRequest, CardGameService_JoinGameServer) error
	ObserveGame(*ObserveGameRequest, CardGameService_ObserveGameServer) error
	GameAction(context.Context, *GameActionRequest) (*Status, error)
	GetGameState(context.Context, *GameStateRequest) (*GameState, error)
	mustEmbedUnimplementedCardGameServiceServer()
}

// UnimplementedCardGameServiceServer must be embedded to have forward compatible implementations.
type UnimplementedCardGameServiceServer struct {
}

func (UnimplementedCardGameServiceServer) Ping(context.Context, *PingRequest) (*PingResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedCardGameServiceServer) Register(*RegisterRequest, CardGameService_RegisterServer) error {
	return status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (UnimplementedCardGameServiceServer) CreateGame(context.Context, *CreateGameRequest) (*CreateGameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateGame not implemented")
}
func (UnimplementedCardGameServiceServer) ListGames(context.Context, *ListGamesRequest) (*ListGamesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListGames not implemented")
}
func (UnimplementedCardGameServiceServer) JoinGame(*JoinGameRequest, CardGameService_JoinGameServer) error {
	return status.Errorf(codes.Unimplemented, "method JoinGame not implemented")
}
func (UnimplementedCardGameServiceServer) ObserveGame(*ObserveGameRequest, CardGameService_ObserveGameServer) error {
	return status.Errorf(codes.Unimplemented, "method ObserveGame not implemented")
}
func (UnimplementedCardGameServiceServer) GameAction(context.Context, *GameActionRequest) (*Status, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GameAction not implemented")
}
func (UnimplementedCardGameServiceServer) GetGameState(context.Context, *GameStateRequest) (*GameState, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetGameState not implemented")
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

func _CardGameService_Register_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(RegisterRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CardGameServiceServer).Register(m, &cardGameServiceRegisterServer{stream})
}

type CardGameService_RegisterServer interface {
	Send(*RegistryActivity) error
	grpc.ServerStream
}

type cardGameServiceRegisterServer struct {
	grpc.ServerStream
}

func (x *cardGameServiceRegisterServer) Send(m *RegistryActivity) error {
	return x.ServerStream.SendMsg(m)
}

func _CardGameService_CreateGame_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateGameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardGameServiceServer).CreateGame(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cards.proto.CardGameService/CreateGame",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardGameServiceServer).CreateGame(ctx, req.(*CreateGameRequest))
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

func _CardGameService_JoinGame_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(JoinGameRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CardGameServiceServer).JoinGame(m, &cardGameServiceJoinGameServer{stream})
}

type CardGameService_JoinGameServer interface {
	Send(*GameActivity) error
	grpc.ServerStream
}

type cardGameServiceJoinGameServer struct {
	grpc.ServerStream
}

func (x *cardGameServiceJoinGameServer) Send(m *GameActivity) error {
	return x.ServerStream.SendMsg(m)
}

func _CardGameService_ObserveGame_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ObserveGameRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CardGameServiceServer).ObserveGame(m, &cardGameServiceObserveGameServer{stream})
}

type CardGameService_ObserveGameServer interface {
	Send(*GameActivity) error
	grpc.ServerStream
}

type cardGameServiceObserveGameServer struct {
	grpc.ServerStream
}

func (x *cardGameServiceObserveGameServer) Send(m *GameActivity) error {
	return x.ServerStream.SendMsg(m)
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
			MethodName: "CreateGame",
			Handler:    _CardGameService_CreateGame_Handler,
		},
		{
			MethodName: "ListGames",
			Handler:    _CardGameService_ListGames_Handler,
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
			StreamName:    "Register",
			Handler:       _CardGameService_Register_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "JoinGame",
			Handler:       _CardGameService_JoinGame_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "ObserveGame",
			Handler:       _CardGameService_ObserveGame_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "game.proto",
}
