package model

type UserInfo struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Account  string `json:"account"`
}

type SubscriptionInfo struct {
	ChatId        int    `json:"chatId"`
	UserId        int    `json:"userId"`
	LastSeenState string `json:"lastSeenState"`
}
