package model

//UserInfo struct to store credentials and subscriptions
type UserInfo struct {
	Login         string                      `json:"login"`
	Password      string                      `json:"password"`
	Subscriptions map[string]SubscriptionInfo `json:"subscriptions,omitempty"`
}

//SubscriptionInfo stores state and chat to notify when changes occur
type SubscriptionInfo struct {
	ChatID        int    `json:"chatId"`
	LastSeenState string `json:"lastSeenState"`
}
