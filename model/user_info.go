package model

type UserInfo struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Account  string `json:"account"`
	Notify   bool   `json:"notify"`
}

type SubscriptionInfo struct {
	ChatId int    `json:"chatId"`
	UserId string `json:"userId"`
}
