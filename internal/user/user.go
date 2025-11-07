package user

type Attributes struct {
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Phone   string   `json:"phone"`
	Address string   `json:"address"`
	Bio     string   `json:"bio"`
	Tags    []string `json:"tags"`
	Avatar  []byte   `json:"avatar"`
}

type User struct {
	ID string `json:"id"`
	Attributes
}
