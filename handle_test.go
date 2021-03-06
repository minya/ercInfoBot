package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/minya/erc/erclib"
	"github.com/minya/ercInfoBot/model"
	"github.com/minya/telegram"
)

func TestHandleReceiptReturnsDocumentIfThereIsTheOnlyAccount(t *testing.T) {
	upd := makeMsgUpdate("/receipt")
	var makeClient = func(l string, p string) ercclient {
		return createFakeERCClient(1)
	}
	h := createHandler(createFakeStorage(), makeClient)
	reply := h.handle(upd)
	ensureDocumentWithButtons(t, reply)
}

func TestHandleGetReturnsResultIfThereIsTheOnlyAccount(t *testing.T) {
	upd := makeMsgUpdate("/get")
	var makeClient = func(l string, p string) ercclient {
		return createFakeERCClient(1)
	}
	h := createHandler(createFakeStorage(), makeClient)
	reply := h.handle(upd)
	ensureMessageWithButtons(t, reply)
}

func TestHandleReturnsChoiceIfMultipleAccountsAndNoAccountSpecified(t *testing.T) {
	var doTest = func(t *testing.T, cmd string) {
		upd := makeMsgUpdate("/get")
		var makeClient = func(l string, p string) ercclient {
			return createFakeERCClient(2)
		}
		h := createHandler(createFakeStorage(), makeClient)
		reply := h.handle(upd).(telegram.ReplyMessage)
		_ = reply.ReplyMarkup.(telegram.InlineKeyboardMarkup)
	}

	for _, cmd := range []string{"/get", "/receipt", "/notify"} {
		doTest(t, cmd)
	}
}

func TestHandleGetReturnsResultIfAccountIsPassedInMessage(t *testing.T) {
	upd := makeMsgUpdate("/get account_0")
	var makeClient = func(l string, p string) ercclient {
		return createFakeERCClient(2)
	}
	h := createHandler(createFakeStorage(), makeClient)
	reply := h.handle(upd)
	ensureMessageWithButtons(t, reply)
}

func TestHandleGetReturnsResultIfAccountIsPassedInCallback(t *testing.T) {
	upd := makeCallbackUpdate("/get account_0")
	var makeClient = func(l string, p string) ercclient {
		return createFakeERCClient(2)
	}
	h := createHandler(createFakeStorage(), makeClient)
	reply := h.handle(upd)
	ensureMessageWithButtons(t, reply)
}

func TestHandleReceiptReturnsDocumentIfAccountIsPassedInMessage(t *testing.T) {
	var makeClient = func(l string, p string) ercclient {
		return createFakeERCClient(2)
	}
	h := createHandler(createFakeStorage(), makeClient)
	reply := h.handle(makeMsgUpdate("/receipt account_0"))
	ensureDocumentWithButtons(t, reply)
}

func TestHandleReceiptReturnsDocumentIfAccountIsPassedInCallback(t *testing.T) {
	var makeClient = func(l string, p string) ercclient {
		return createFakeERCClient(2)
	}
	h := createHandler(createFakeStorage(), makeClient)
	reply := h.handle(makeCallbackUpdate("/receipt account_0"))
	ensureDocumentWithButtons(t, reply)
}

func TestNotifyWritesDataToStorage(t *testing.T) {
	var doTest = func(t *testing.T, upd telegram.Update, numAccounts uint) {
		var makeClient = func(l string, p string) ercclient {
			return createFakeERCClient(numAccounts)
		}
		userWritten := false
		var onUserSave = func(savingUserID int, user model.UserInfo) {
			if savingUserID == 0 {
				t.Error("UserID must not be 0")
			}
			if savingUserID != userID {
				t.Error("UserID mismatch")
			}
			if user.Subscriptions["account_0"].ChatID != chatID {
				t.Error("Subscription chatID mismatch")
			}
			userWritten = true
		}
		h := createHandler(createFakeStorageCapturingWrites(onUserSave), makeClient)
		reply := h.handle(upd)
		_ = reply.(telegram.ReplyMessage)
		if !userWritten {
			t.Error("User was never written")
		}
	}

	doTest(t, makeCallbackUpdate("/notify account_0"), 2)
	doTest(t, makeMsgUpdate("/notify account_0"), 2)
	doTest(t, makeMsgUpdate("/notify"), 1)
}

func ensureDocumentWithButtons(t *testing.T, reply interface{}) {
	doc := reply.(telegram.ReplyDocument)
	_ = doc.ReplyMarkup.(telegram.ReplyKeyboardMarkup)
}

func ensureMessageWithButtons(t *testing.T, reply interface{}) {
	msg := reply.(telegram.ReplyMessage)
	_ = msg.ReplyMarkup.(telegram.ReplyKeyboardMarkup)
}

func makeMsgUpdate(commandText string) telegram.Update {
	return telegram.Update{
		UpdateId: 431,
		Message:  makeMessage(commandText),
	}
}

func makeCallbackUpdate(commandText string) telegram.Update {
	return telegram.Update{
		UpdateId: 431,
		CallbackQuery: telegram.CallbackQuery{
			Id:      "12123",
			Message: makeMessage("OLOLO"),
			Data:    commandText,
			From:    makeUser(),
		},
	}
}

func makeMessage(text string) telegram.Message {
	return telegram.Message{
		MessageId: 1,
		From:      makeUser(),
		Date:      time.Now().Unix(),
		Chat: telegram.Chat{
			Id:       chatID,
			Type:     "private",
			Title:    "title",
			Username: userName,
		},
		Text: text,
	}
}

func makeUser() telegram.User {
	return telegram.User{
		Id:       userID,
		UserName: userName,
	}
}

var userID = 100500
var userName = "@ololo"
var chatID = 404040

type fakeStorage struct {
	userInfo model.UserInfo
	onWrite  func(int, model.UserInfo)
}

func (s fakeStorage) GetUserInfo(userID int) (model.UserInfo, error) {
	return s.userInfo, nil
}

func (s fakeStorage) SaveUser(userID int, userInfo model.UserInfo) error {
	if s.onWrite != nil {
		s.onWrite(userID, userInfo)
	}
	return nil
}
func (s fakeStorage) GetUsers() (map[int]model.UserInfo, error) {
	return map[int]model.UserInfo{123: s.userInfo}, nil
}

type fakeERCClient struct {
	accounts []erclib.Account
}

func (f fakeERCClient) GetAccounts() ([]erclib.Account, error) {
	return f.accounts, nil
}

func (f fakeERCClient) GetBalanceInfo(account string, t time.Time) (erclib.BalanceInfo, error) {
	balance := erclib.BalanceInfo{
		Month:    "Январь",
		Credit:   erclib.Details{},
		Debit:    erclib.Details{},
		AtTheEnd: erclib.Details{},
	}
	return balance, nil
}

func (f fakeERCClient) GetReceipt(accNumber string) ([]byte, error) {
	return []byte{1}, nil
}

func createFakeERCClient(numAccounts uint) fakeERCClient {
	result := make([]erclib.Account, numAccounts, numAccounts)
	var i uint
	for i = 0; i < numAccounts; i++ {
		result[i] = erclib.Account{
			Number:  fmt.Sprintf("account_%v", i),
			Address: fmt.Sprintf("Address %v", i),
		}
	}
	return fakeERCClient{accounts: result}
}

func createFakeStorage() fakeStorage {
	return fakeStorage{userInfo: model.UserInfo{Login: "login@gmail.com"}}
}

func createFakeStorageCapturingWrites(onWrite func(int, model.UserInfo)) fakeStorage {
	return fakeStorage{userInfo: model.UserInfo{Login: "login@gmail.com"}, onWrite: onWrite}
}
