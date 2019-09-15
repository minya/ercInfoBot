package model

//UserStorage is to store users
type UserStorage interface {
	GetUserInfo(userID int) (UserInfo, error)
	SaveUser(userID int, userInfo UserInfo) error
	GetUsers() (map[int]UserInfo, error)
}
