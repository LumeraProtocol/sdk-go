package grpc

import (
	"context"
	"fmt"

	"github.com/cosmos/gogoproto/proto"
	googlegrpc "google.golang.org/grpc"
)

type gogoCodec struct{}
type gogoClientConn struct {
	conn googlegrpc.ClientConnInterface
}

// GogoCodec returns a gRPC codec for gogoproto-generated Cosmos SDK messages.
func GogoCodec() gogoCodec {
	return gogoCodec{}
}

func (gogoCodec) Name() string {
	return "proto"
}

func (gogoCodec) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("gogo codec expects proto.Message, got %T", v)
	}
	return proto.Marshal(msg)
}

func (gogoCodec) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("gogo codec expects proto.Message, got %T", v)
	}
	return proto.Unmarshal(data, msg)
}

// GogoClientConn forces the gogo codec for gogoproto-generated query clients.
func GogoClientConn(conn googlegrpc.ClientConnInterface) googlegrpc.ClientConnInterface {
	return gogoClientConn{conn: conn}
}

func (c gogoClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...googlegrpc.CallOption) error {
	opts = append([]googlegrpc.CallOption{googlegrpc.ForceCodec(GogoCodec())}, opts...)
	return c.conn.Invoke(ctx, method, args, reply, opts...)
}

func (c gogoClientConn) NewStream(ctx context.Context, desc *googlegrpc.StreamDesc, method string, opts ...googlegrpc.CallOption) (googlegrpc.ClientStream, error) {
	opts = append([]googlegrpc.CallOption{googlegrpc.ForceCodec(GogoCodec())}, opts...)
	return c.conn.NewStream(ctx, desc, method, opts...)
}
