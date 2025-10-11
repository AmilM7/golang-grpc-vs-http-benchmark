package user

// User represents a user entity exposed by both HTTP and gRPC transports.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}
