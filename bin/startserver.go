package main

import (
	"fmt"
	"time"

	"github.com/grokify/gotilla/fmt/fmtutil"
	"github.com/joho/godotenv"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"

	"github.com/grokify/alexa-skill-ringcentral-go"
	"github.com/grokify/alexa-skill-ringcentral-go/src/config"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg, err := config.NewConfiguration()
	if err != nil {
		fmt.Printf("Error [%v]\n", err)
		panic("E_CONFIG_FAILURE")
	}
	cfg.Port = 3000
	cfg.LogLevel = log.DebugLevel

	fmtutil.PrintJSON(cfg)

	cfg.Cache = cache.New(5*time.Minute, 10*time.Minute)

	rcskillserver.StartServer(cfg)
}
