package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/minya/erc/erclib"
	"github.com/minya/goutils/config"
	"github.com/minya/telegramInfoBot/core"
	"github.com/minya/telegramInfoBot/model"
)

var makeERCClient = func(l string, p string) erclib.ErcClient {
	return erclib.NewErcClientWithCredentials(l, p)
}

func main() {
	settings, storage := initialize()
	core.Run(settings, storage, Process, ProcessNotification)
}

func initialize() (BotSettings, model.FirebaseStorage) {
	var settings BotSettings
	var storage model.FirebaseStorage
	var logPath string
	flag.StringVar(&logPath, "logpath", "ercInfoBot.log", "Path to write logs")
	var configPath string
	flag.StringVar(&configPath, "cfg", "~/.ercInfoBot/settings.json", "Path to write logs")
	flag.Parse()
	setUpLogger(logPath)

	errCfg := config.UnmarshalJson(&settings, configPath)

	if nil != errCfg {
		panic(fmt.Sprintf("Unable to read settings from '%v': %v \n", configPath, errCfg))
	}
	log.Printf("Config read: %v\n", settings)

	if !settings.IsValid() {
		log.Fatalf("Incorrect settings: %v\n", settings)
		panic("Incorrect settings")
	}

	storageSettings := settings.StorageSettings
	storage = model.NewFirebaseStorage(
		storageSettings.BaseURL,
		storageSettings.APIKey,
		storageSettings.Login,
		storageSettings.Password)
	return settings, storage
}

// BotSettings struct to represent stored settings
type BotSettings struct {
	ID                string                 `json:"id"`
	UpdateCheckPeriod string                 `json:"updateCheckPeriod"`
	StorageSettings   model.FirebaseSettings `json:"storageSettings"`
}

func (settings BotSettings) GetBotToken() string {
	return settings.ID
}

func (settings BotSettings) GetNotifierSettings() core.NotifierSettings {
	var notifierSettings core.NotifierSettings
	var errParseDuration error
	updateCheckPeriod, errParseDuration := time.ParseDuration(settings.UpdateCheckPeriod)
	if errParseDuration != nil {
		log.Fatalf("Unable to parse duration from '%v': %v \n", settings.UpdateCheckPeriod, errParseDuration)
	}
	notifierSettings.UpdateCheckPeriod = updateCheckPeriod
	return notifierSettings
}

func (theSettings BotSettings) IsValid() bool {
	fbSettings := &theSettings.StorageSettings
	return theSettings.ID != "" &&
		fbSettings.APIKey != "" &&
		fbSettings.BaseURL != "" &&
		fbSettings.Login != "" &&
		fbSettings.Password != "" &&
		theSettings.UpdateCheckPeriod != ""
}

func setUpLogger(logPath string) {
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(logFile)
}
