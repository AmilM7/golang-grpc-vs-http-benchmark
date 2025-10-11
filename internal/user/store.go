package user

import (
	"errors"
	"sort"
	"strconv"
	"sync"
)

var (
	// ErrNotFound indicates that the requested user does not exist in the store.
	ErrNotFound = errors.New("user not found")
)

// Store is a concurrency-safe, in-memory repository for users.
type Store struct {
	mu     sync.RWMutex
	users  map[string]User
	nextID int64
}

// NewStore constructs an empty Store instance.
func NewStore() *Store {
	return &Store{
		users: make(map[string]User),
	}
}

// Create persists a new user and returns the created entity.
func (s *Store) Create(name, email string) User {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	id := strconv.FormatInt(s.nextID, 10)
	u := User{
		ID:    id,
		Name:  name,
		Email: email,
	}
	s.users[id] = u
	return u
}

// Update modifies an existing user. ErrNotFound is returned if the user does not exist.
func (s *Store) Update(id, name, email string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	u := User{
		ID:    id,
		Name:  name,
		Email: email,
	}
	s.users[id] = u
	return u, nil
}

// Get retrieves a user by id.
func (s *Store) Get(id string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[id]
	return u, ok
}

// Delete removes a user by id. It returns true if the user existed.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[id]; !ok {
		return false
	}
	delete(s.users, id)
	return true
}

// List returns all users sorted by identifier.
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
