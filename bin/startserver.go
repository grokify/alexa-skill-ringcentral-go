package main

import (
	"context"
	"fmt"
	"time"

	"github.com/grokify/mogo/fmt/fmtutil"
	"github.com/joho/godotenv"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"

	rcskillserver "github.com/grokify/alexa-skill-ringcentral-go"
	"github.com/grokify/alexa-skill-ringcentral-go/src/config"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
	}

	cfg, err := config.NewConfiguration(context.Background())
	if err != nil {
		fmt.Printf("error [%v]\n", err)
		panic("error config failure")
	}
	cfg.Port = 3000
	cfg.LogLevel = log.DebugLevel

	fmtutil.MustPrintJSON(cfg)

	cfg.Cache = cache.New(5*time.Minute, 10*time.Minute)

	rcskillserver.StartServer(cfg)
}
