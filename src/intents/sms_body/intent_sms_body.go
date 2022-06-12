package smsbody

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/grokify/mogo/net/httputilmore"
	"github.com/grokify/ringcentral-sdk-go/rcsdk/definitions"
	rchttp "github.com/grokify/ringcentral-sdk-go/rcsdk/http"
	"github.com/grokify/ringcentral-sdk-go/rcsdk/requests"
	alexa "github.com/mikeflynn/go-alexa/skillserver"
	log "github.com/sirupsen/logrus"

	"github.com/grokify/alexa-skill-ringcentral-go/src/config"
	"github.com/grokify/alexa-skill-ringcentral-go/src/intents/sms"
)

const (
	GenericError string = "RingCentral skill ran into an error processing your request."
)

func GetSessionData(cfg config.Configuration, sessionID string) (sms.SessionInfoSMS, error) {
	alexaSessionData := sms.SessionInfoSMS{}

	alexaSessionInterface, found := cfg.Cache.Get(sessionID)
	if !found {
		return alexaSessionData, errors.New("SessionID Not Found")
	}
	alexaSessionString := alexaSessionInterface.(string)

	err := json.Unmarshal([]byte(alexaSessionString), &alexaSessionData)
	return alexaSessionData, err
}

func HandleRequest(cfg config.Configuration, echoReq *alexa.EchoRequest) *alexa.EchoResponse {
	alexaSessionData, err := GetSessionData(cfg, echoReq.Session.SessionID)
	if err != nil {
		return IntentErrorResponse(GenericError)
	}

	messageText := echoReq.Request.Intent.Slots["MessageText"].Value

	if len(strings.TrimSpace(messageText)) == 0 {
		return IntentErrorResponse(GenericError)
	}

	contact := alexaSessionData.Contact
	if len(strings.TrimSpace(contact.PhoneNumber)) < 1 {
		return IntentErrorResponse(fmt.Sprintf("I couldn't not find a phone number for %s", contact.FullName()))
	}

	if len(strings.TrimSpace(messageText)) < 1 {
		return alexa.NewEchoResponse().OutputSpeech(
			"Please say your message").EndSession(false)
	}

	myNumber := cfg.RcPhoneNumberSMS
	rcReqBody := requests.AccountExtensionSmsPostRequestBody{
		Text: messageText,
		To:   []definitions.CallerInfo{{PhoneNumber: contact.PhoneNumber}},
		From: definitions.CallerInfo{PhoneNumber: myNumber}}

	rcReqBodyBytes, err := json.Marshal(rcReqBody)

	if err != nil {
		log.WithFields(log.Fields{
			"type":   "rc.request",
			"status": "json.encode.error"}).Warn(fmt.Sprintf("Error: %v\n", err))
	} else {
		log.WithFields(log.Fields{
			"type": "rc.request",
			"item": "request.body"}).Debug(string(rcReqBodyBytes))
	}

	rcReq := rchttp.Request2{
		Method:  http.MethodPost,
		URL:     "/account/~/extension/~/sms",
		Headers: http.Header{},
		Body:    bytes.NewReader(rcReqBodyBytes)}
	rcReq.Headers.Add(httputilmore.HeaderContentType, httputilmore.ContentTypeAppJSONUtf8)

	rcResp, err := cfg.Platform.APICall(rcReq)
	if err != nil || rcResp.StatusCode >= 400 {
		log.WithFields(log.Fields{
			"type":   "rc.api.response",
			"error":  err.Error()}).
			Warn("API response failure")
		return IntentErrorResponse(fmt.Sprintf("An error occurred calling %v", contact.FullName()))
	}

	rcRespBody, err := io.ReadAll(rcResp.Body)

	log.WithFields(log.Fields{
		"type":   "rc.response",
		"status": "status.code"}).
		Info(
			fmt.Sprintf("Status Code: %v\n", rcResp.StatusCode))

	if err != nil || rcResp.StatusCode >= 400 {
		log.WithFields(log.Fields{
			"type":   "rc.response",
			"status": "json.encode.error",
			"error":  fmt.Sprintf("%v", err)}).
			Warn(
				fmt.Sprintf("Error: %v\n", string(rcRespBody)))
		return IntentErrorResponse(fmt.Sprintf("An error occurred calling %v", contact.FullName()))
	}

	log.WithFields(log.Fields{
		"type":   "rc.response",
		"status": "response.body"}).
		Info(fmt.Sprintf("SuccessBody: %v\n", string(rcRespBody)))

	actionText := fmt.Sprintf("Text sent to %s. %s", contact.FullName(), messageText)

	return alexa.NewEchoResponse().OutputSpeech(
		actionText).Card(
		"Alexa RingCentral", actionText).EndSession(true)
}

func IntentErrorResponse(s string) *alexa.EchoResponse {
	return alexa.NewEchoResponse().OutputSpeech(s).Card(
		"Alexa RingCentral", s).EndSession(true)
}
