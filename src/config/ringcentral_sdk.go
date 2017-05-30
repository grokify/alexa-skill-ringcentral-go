package config

import (
	"github.com/grokify/ringcentral-sdk-go"
	"github.com/grokify/ringcentral-sdk-go/rcsdk/platform"
)

func GetRingCentralSdk(cfg Configuration) (platform.Platform, error) {
	sdk := rcsdk.NewSdk(cfg.RcAppKey, cfg.RcAppSecret, cfg.RcServerURL)
	platform := sdk.GetPlatform()
	_, err := platform.Authorize(cfg.RcUsername, cfg.RcExtension, cfg.RcPassword, false)
	return platform, err
}
