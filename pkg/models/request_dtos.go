package models

type loginReq struct {
	LoginOrEmail string `json:"login_or_email"`
	Password     string `json:"password"`
}
