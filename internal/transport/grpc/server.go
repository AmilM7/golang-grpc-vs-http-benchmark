package grpctransport

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	userpb "golang-grpc/pkg/gen/user/v1"
)

// NewServer constructs a gRPC server and registers the user service.
func NewServer(svc userpb.UserServiceServer) *grpc.Server {
	server := grpc.NewServer()
	userpb.RegisterUserServiceServer(server, svc)
	reflection.Register(server)
	return server
}
