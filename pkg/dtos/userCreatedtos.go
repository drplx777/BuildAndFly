package dtos

type Registerdtos struct {
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}
