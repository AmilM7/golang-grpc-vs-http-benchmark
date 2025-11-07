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
	ErrInvalidInput = errors.New("invalid user input")
)

type Service struct {
	store *user.Store
	userpb.UnimplementedUserServiceServer
}

func NewUserService(store *user.Store) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) Create(_ context.Context, attrs user.Attributes) (user.User, error) {
	clean := normalizeAttributes(attrs)
	if err := validatePayload(clean); err != nil {
		return user.User{}, err
	}
	return s.store.Create(clean), nil
}

func (s *Service) Update(_ context.Context, id string, attrs user.Attributes) (user.User, error) {
	if err := validateIdentifier(id); err != nil {
		return user.User{}, err
	}
	clean := normalizeAttributes(attrs)
	if err := validatePayload(clean); err != nil {
		return user.User{}, err
	}
	updated, err := s.store.Update(id, clean)
	if err != nil {
		return user.User{}, err
	}
	return updated, nil
}

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

func (s *Service) Delete(_ context.Context, id string) error {
	if err := validateIdentifier(id); err != nil {
		return err
	}
	if ok := s.store.Delete(id); !ok {
		return user.ErrNotFound
	}
	return nil
}

func (s *Service) List(_ context.Context) []user.User {
	return s.store.List()
}

func (s *Service) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.UserResponse, error) {
	attrs := protoToAttributes(req.GetName(), req.GetEmail(), req.GetPhone(), req.GetAddress(), req.GetBio(), req.GetTags(), req.GetAvatar())
	u, err := s.Create(ctx, attrs)
	if err != nil {
		return nil, serviceError(err)
	}
	return &userpb.UserResponse{User: toProto(u)}, nil
}

func (s *Service) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.UserResponse, error) {
	u, err := s.Get(ctx, strings.TrimSpace(req.GetId()))
	if err != nil {
		return nil, serviceError(err)
	}
	return &userpb.UserResponse{User: toProto(u)}, nil
}

func (s *Service) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UserResponse, error) {
	attrs := protoToAttributes(req.GetName(), req.GetEmail(), req.GetPhone(), req.GetAddress(), req.GetBio(), req.GetTags(), req.GetAvatar())
	u, err := s.Update(ctx, strings.TrimSpace(req.GetId()), attrs)
	if err != nil {
		return nil, serviceError(err)
	}
	return &userpb.UserResponse{User: toProto(u)}, nil
}

func (s *Service) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*emptypb.Empty, error) {
	if err := s.Delete(ctx, strings.TrimSpace(req.GetId())); err != nil {
		return nil, serviceError(err)
	}
	return &emptypb.Empty{}, nil
}

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
		Id:      u.ID,
		Name:    u.Name,
		Email:   u.Email,
		Phone:   u.Phone,
		Address: u.Address,
		Bio:     u.Bio,
		Tags:    append([]string(nil), u.Tags...),
		Avatar:  append([]byte(nil), u.Avatar...),
	}
}

func validatePayload(attrs user.Attributes) error {
	if attrs.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if attrs.Email == "" || !strings.Contains(attrs.Email, "@") {
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

func protoToAttributes(name, email, phone, address, bio string, tags []string, avatar []byte) user.Attributes {
	return user.Attributes{
		Name:    name,
		Email:   email,
		Phone:   phone,
		Address: address,
		Bio:     bio,
		Tags:    append([]string(nil), tags...),
		Avatar:  append([]byte(nil), avatar...),
	}
}

func normalizeAttributes(attrs user.Attributes) user.Attributes {
	attrs.Name = strings.TrimSpace(attrs.Name)
	attrs.Email = strings.TrimSpace(attrs.Email)
	attrs.Phone = strings.TrimSpace(attrs.Phone)
	attrs.Address = strings.TrimSpace(attrs.Address)
	attrs.Bio = strings.TrimSpace(attrs.Bio)
	if len(attrs.Tags) > 0 {
		clean := attrs.Tags[:0]
		for _, tag := range attrs.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				clean = append(clean, tag)
			}
		}
		attrs.Tags = clean
	}
	return attrs
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
