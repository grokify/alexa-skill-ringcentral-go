package sms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/grokify/mogo/net/httputilmore"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"

	"github.com/grokify/ringcentral-sdk-go/rcsdk/definitions"
	rchttp "github.com/grokify/ringcentral-sdk-go/rcsdk/http"
	"github.com/grokify/ringcentral-sdk-go/rcsdk/requests"
	alexa "github.com/mikeflynn/go-alexa/skillserver"

	"github.com/grokify/alexa-skill-ringcentral-go/src/config"
)

func HandleRequest(cfg config.Configuration, echoReq *alexa.EchoRequest) *alexa.EchoResponse {
	// var echoResp  alexa.NewEchoResponse()

	intent := echoReq.Request.Intent
	firstName := intent.Slots["FirstName"].Value
	messageText := intent.Slots["MessageText"].Value

	contact, err := cfg.AddressBook.GetContactByFirstName(firstName)
	if err != nil {
		log.WithFields(log.Fields{
			"type":   "addressbook.contact",
			"status": "contact not found",
		}).Info(fmt.Sprintf("%v not found.", firstName))
		return IntentErrorResponse(fmt.Sprintf("I'm sorry but I couldn't find  %v", firstName))
	}

	log.WithFields(log.Fields{
		"type": "contact.info"}).Debug(fmt.Sprintf("Contact: %v\n", contact))

	toPhoneNumber := contact.PhoneNumber
	if len(contact.PhoneNumber) > 0 {
		if len(strings.TrimSpace(messageText)) < 1 {
			sessionInfoSMS := SessionInfoSMS{
				AlexaSessionID: echoReq.Session.SessionID,
				Contact:        contact,
				Intent:         "RingCentralSendSMSIntent"}
			sessionInfoSMSBytes, err := json.Marshal(sessionInfoSMS)
			if err != nil {
				return IntentErrorResponse("The RingCentral skill has encountered an error.")
			}
			cfg.Cache.Set(echoReq.Session.SessionID,
				string(sessionInfoSMSBytes),
				cache.DefaultExpiration)

			return alexa.NewEchoResponse().OutputSpeech(
				fmt.Sprintf("Sending message to %s. Please say your message starting with message body", contact.FullName())).EndSession(false)
		}

		myNumber := cfg.RcPhoneNumberSMS
		reqBody := requests.AccountExtensionSmsPostRequestBody{
			Text: messageText,
			To:   []definitions.CallerInfo{{PhoneNumber: toPhoneNumber}},
			From: definitions.CallerInfo{PhoneNumber: myNumber}}

		reqBytes, err := json.Marshal(reqBody)
		if err != nil {
			log.WithFields(log.Fields{
				"type":   "json.encode.error",
				"status": "error",
			}).Info(fmt.Sprintf("Error: %v\n", err))
		} else {
			log.WithFields(log.Fields{
				"type":   "rc.request.body",
				"status": "info",
			}).Info(string(reqBytes))
		}

		req2 := rchttp.Request2{
			Method:  "post",
			URL:     "/account/~/extension/~/sms",
			Headers: http.Header{},
			Body:    bytes.NewReader(reqBytes)}
		req2.Headers.Add(httputilmore.HeaderContentType, httputilmore.ContentTypeAppJSONUtf8)

		resp, err := cfg.Platform.APICall(req2)
		if err != nil || resp.StatusCode >= 400 {
			log.WithFields(log.Fields{
				"type":   "rc.response",
				"status": "json.encode.error",
				"error":  err.Error()}).
				Warn(
					fmt.Sprintf("Error: %v\n", "error calling extension/sms"))
			return IntentErrorResponse(fmt.Sprintf("an error occurred calling [%v]", "extension/sms"))
		}
		rcRespBody, err := io.ReadAll(resp.Body)
		if err != nil || resp.StatusCode >= 400 {
			log.WithFields(log.Fields{
				"type":   "rc.response",
				"status": "json.encode.error",
				"error":  err.Error()}).
				Warn(
					fmt.Sprintf("Error: %v\n", "error parsing response from extension/sms"))
			return IntentErrorResponse(fmt.Sprintf("an error occurred parsing response from [%v]", "extension/sms"))
		}

		log.WithFields(log.Fields{
			"type": "rc.response.status_code"}).Info(fmt.Sprintf("%v", resp.StatusCode))

		if err != nil || resp.StatusCode >= 400 {
			log.WithFields(log.Fields{
				"type":   "rc.response",
				"status": "json.encode.error",
				"error":  fmt.Sprintf("%v", err)}).
				Warn(
					fmt.Sprintf("Error: %v\n", string(rcRespBody)))
			return IntentErrorResponse(fmt.Sprintf("an error occurred calling %v", contact.FullName()))
		} else {
			log.WithFields(log.Fields{
				"type":   "rc.response",
				"status": "response.body"}).
				Debug(fmt.Sprintf("Body: %v\n", string(rcRespBody)))
		}
	} else {
		return IntentErrorResponse(fmt.Sprintf("couldn't find a number for %v", contact.FullName()))
	}

	actionText := fmt.Sprintf("Calling %s", contact.FullName())

	return alexa.NewEchoResponse().OutputSpeech(
		actionText).Card(
		"Alexa RingCentral", actionText).EndSession(true)
}

func IntentErrorResponse(s string) *alexa.EchoResponse {
	return alexa.NewEchoResponse().OutputSpeech(s).Card(
		"Alexa RingCentral", s).EndSession(true)
}

type SessionInfoSMS struct {
	AlexaSessionID string
	Contact        config.Contact
	Intent         string
}
