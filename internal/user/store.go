package user

import (
	"errors"
	"sort"
	"strconv"
	"sync"
)

var (
	ErrNotFound = errors.New("user not found")
)

type Store struct {
	mu     sync.RWMutex
	users  map[string]User
	nextID int64
}

func NewStore() *Store {
	return &Store{
		users: make(map[string]User),
	}
}

func (s *Store) Create(attrs Attributes) User {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	id := strconv.FormatInt(s.nextID, 10)
	u := User{
		ID:         id,
		Attributes: cloneAttributes(attrs),
	}
	s.users[id] = u
	return u
}

func (s *Store) Update(id string, attrs Attributes) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	u := User{
		ID:         id,
		Attributes: cloneAttributes(attrs),
	}
	s.users[id] = u
	return u, nil
}

func (s *Store) Get(id string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[id]
	return u, ok
}

func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[id]; !ok {
		return false
	}
	delete(s.users, id)
	return true
}

func (s *Store) List() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.users) == 0 {
		return nil
	}

	users := make([]User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].ID < users[j].ID
	})
	return users
}

func cloneAttributes(attrs Attributes) Attributes {
	copied := attrs
	if attrs.Tags != nil {
		copied.Tags = append([]string(nil), attrs.Tags...)
	}
	if attrs.Avatar != nil {
		copied.Avatar = append([]byte(nil), attrs.Avatar...)
	}
	return copied
}
