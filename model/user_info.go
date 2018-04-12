package model

type UserInfo struct {
	Login        string           `json:"login"`
	Password     string           `json:"password"`
	Account      string           `json:"account"`
	Subscription SubscriptionInfo `json:"subscription,omitempty"`
}

type SubscriptionInfo struct {
	ChatId        int    `json:"chatId"`
	LastSeenState string `json:"lastSeenState"`
}
