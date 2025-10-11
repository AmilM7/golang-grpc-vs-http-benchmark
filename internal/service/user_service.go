package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang-grpc/internal/user"
	userpb "golang-grpc/pkg/gen/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	// ErrInvalidInput indicates that the supplied user payload failed validation.
	ErrInvalidInput = errors.New("invalid user input")
)

// Service orchestrates user operations and implements the gRPC contract.
type Service struct {
	store *user.Store
	userpb.UnimplementedUserServiceServer
}

// NewUserService constructs a Service backed by the provided store.
func NewUserService(store *user.Store) *Service {
	return &Service{
		store: store,
	}
}

// Create creates a new user after validating the payload.
func (s *Service) Create(_ context.Context, name, email string) (user.User, error) {
	if err := validatePayload(name, email); err != nil {
		return user.User{}, err
	}
	return s.store.Create(name, email), nil
}

// Update updates an existing user by id.
func (s *Service) Update(_ context.Context, id, name, email string) (user.User, error) {
	if err := validateIdentifier(id); err != nil {
		return user.User{}, err
	}
	if err := validatePayload(name, email); err != nil {
		return user.User{}, err
	}
	updated, err := s.store.Update(id, name, email)
	if err != nil {
		return user.User{}, err
	}
	return updated, nil
}

// Get retrieves a user by id.
func (s *Service) Get(_ context.Context, id string) (user.User, error) {
	if err := validateIdentifier(id); err != nil {
		return user.User{}, err
	}
	u, ok := s.store.Get(id)
	if !ok {
		return user.User{}, user.ErrNotFound
	}
	return u, nil
}

// Delete removes a user by id.
func (s *Service) Delete(_ context.Context, id string) error {
	if err := validateIdentifier(id); err != nil {
		return err
	}
	if ok := s.store.Delete(id); !ok {
		return user.ErrNotFound
	}
	return nil
}

// List returns all users currently persisted.
func (s *Service) List(_ context.Context) []user.User {
	return s.store.List()
}

// CreateUser implements userpb.UserServiceServer.
func (s *Service) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.UserResponse, error) {
	u, err := s.Create(ctx, strings.TrimSpace(req.GetName()), strings.TrimSpace(req.GetEmail()))
	if err != nil {
		return nil, serviceError(err)
	}
	return &userpb.UserResponse{User: toProto(u)}, nil
}

// GetUser implements userpb.UserServiceServer.
func (s *Service) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.UserResponse, error) {
	u, err := s.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, serviceError(err)
	}
	return &userpb.UserResponse{User: toProto(u)}, nil
}

// UpdateUser implements userpb.UserServiceServer.
func (s *Service) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UserResponse, error) {
	u, err := s.Update(ctx, strings.TrimSpace(req.GetId()), strings.TrimSpace(req.GetName()), strings.TrimSpace(req.GetEmail()))
	if err != nil {
		return nil, serviceError(err)
	}
	return &userpb.UserResponse{User: toProto(u)}, nil
}

// DeleteUser implements userpb.UserServiceServer.
func (s *Service) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*emptypb.Empty, error) {
	if err := s.Delete(ctx, strings.TrimSpace(req.GetId())); err != nil {
		return nil, serviceError(err)
	}
	return &emptypb.Empty{}, nil
}

// ListUsers implements userpb.UserServiceServer.
func (s *Service) ListUsers(ctx context.Context, _ *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	users := s.List(ctx)
	resp := &userpb.ListUsersResponse{
		Users: make([]*userpb.User, 0, len(users)),
	}
	for _, u := range users {
		resp.Users = append(resp.Users, toProto(u))
	}
	return resp, nil
}

func toProto(u user.User) *userpb.User {
	return &userpb.User{
		Id:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}
}

func validatePayload(name, email string) error {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("%w: email must contain '@'", ErrInvalidInput)
	}
	return nil
}

func validateIdentifier(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidInput)
	}
	return nil
}

func serviceError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, user.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
