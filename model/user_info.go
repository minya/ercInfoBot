package model

type UserInfo struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Account  string `json:"account"`
	Notify   bool   `json:"notify"`
}
