package model

import (
	"github.com/melvinmt/firebase"
	"github.com/minya/googleapis"
	"strconv"
)

type FirebaseStorage struct {
	ApiKey   string
	Login    string
	Password string
	BaseUrl  string
}

func NewFirebaseStorage(baseUrl string, apiKey string, login string, password string) FirebaseStorage {
	var storage FirebaseStorage
	storage.BaseUrl = baseUrl
	storage.ApiKey = apiKey
	storage.Login = login
	storage.Password = password
	return storage
}

func (this FirebaseStorage) GetUserInfo(userId int) (UserInfo, error) {
	ref, err := this.getUserReference(strconv.Itoa(userId))
	var result UserInfo
	if err = ref.Value(&result); err != nil {
		return result, err
	}
	return result, nil
}

func (this FirebaseStorage) SetUserInfo(userId string, userInfo UserInfo) error {
	ref, err := this.getUserReference(userId)
	if err = ref.Write(userInfo); err != nil {
		return err
	}
	return nil
}

func (this FirebaseStorage) GetSubscriptions() (map[string]SubscriptionInfo, error) {
	ref, err := this.getReference("/subscriptions")
	if err != nil {
		return nil, err
	}

	var subsMap map[string]SubscriptionInfo
	ref.Value(&subsMap)
	return subsMap, nil
}

func (this FirebaseStorage) SaveSubscription(id string, s SubscriptionInfo) error {
	ref, err := this.getReference("/subscriptions/" + id)
	if nil != err {
		return err
	}
	return ref.Write(s)
}

func (this FirebaseStorage) getUserReference(userId string) (*firebase.Reference, error) {
	return this.getReference("/accounts/" + userId)
}

func (this FirebaseStorage) getReference(path string) (*firebase.Reference, error) {
	idToken, err := this.signIn()
	if nil != err {
		return nil, err
	}
	base := this.BaseUrl
	ref := firebase.NewReference(base + path).Auth(idToken)
	return ref, nil
}

func (this FirebaseStorage) signIn() (string, error) {
	response, err := googleapis.SignInWithEmailAndPassword(
		this.Login, this.Password, this.ApiKey)
	if nil != err {
		return "", err
	}
	return response.IdToken, nil
}
