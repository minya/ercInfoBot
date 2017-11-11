package model

import (
	"github.com/melvinmt/firebase"
	"github.com/minya/googleapis"
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

func GetUserInfo(this *FirebaseStorage, userId string) (UserInfo, error) {
	ref, err := getReference(this, userId)
	var result UserInfo
	if err = ref.Value(&result); err != nil {
		return result, err
	}
	return result, nil
}

func SetUserInfo(this *FirebaseStorage, userId string, userInfo UserInfo) error {
	ref, err := getReference(this, userId)
	if err = ref.Write(userInfo); err != nil {
		return err
	}
	return nil
}

func getReference(this *FirebaseStorage, userId string) (*firebase.Reference, error) {
	//token := "AIzaSyCd2EINByhPwPz-gqpX3QGYx3Wr2FA4dgg"
	response, err := googleapis.SignInWithEmailAndPassword(
		this.Login, this.Password, this.ApiKey)
	if err != nil {
		return nil, err
	}

	base := this.BaseUrl
	ref := firebase.NewReference(base + "/accounts/" + userId).Auth(response.IdToken)
	return ref, nil
}
