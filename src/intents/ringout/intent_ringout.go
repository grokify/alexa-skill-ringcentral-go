package ringout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/grokify/mogo/net/http/httpsimple"
	"github.com/grokify/mogo/net/http/httputilmore"
	"github.com/grokify/ringcentral-sdk-go/rcsdk/definitions"
	"github.com/grokify/ringcentral-sdk-go/rcsdk/requests"

	alexa "github.com/mikeflynn/go-alexa/skillserver"
	log "github.com/sirupsen/logrus"

	"github.com/grokify/alexa-skill-ringcentral-go/src/config"
)

func HandleRequest(ctx context.Context, cfg config.Configuration, echoReq *alexa.EchoRequest) *alexa.EchoResponse {
	intent := echoReq.Request.Intent
	firstName := intent.Slots["FirstName"].Value

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
		myNumber := cfg.RcPhoneNumberRingOut
		reqBody := requests.AccountExtensionRingoutPostRequestBody{
			CallerId:   definitions.RingOut_Request_To{PhoneNumber: myNumber},
			To:         definitions.RingOut_Request_To{PhoneNumber: toPhoneNumber},
			From:       definitions.RingOut_Request_From{PhoneNumber: myNumber},
			PlayPrompt: false}

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

		rcReq := httpsimple.Request{
			Method:  http.MethodPost,
			URL:     "/account/~/extension/~/ringout",
			Headers: http.Header{},
			Body:    bytes.NewReader(reqBytes)}
		rcReq.Headers.Add(httputilmore.HeaderContentType, httputilmore.ContentTypeAppJSONUtf8)

		resp, err := cfg.Platform.APICall(ctx, rcReq)
		if err != nil {
			log.WithFields(log.Fields{
				"type":  "http request error",
				"error": err.Error(),
			}).Warn("ringout API call failed")
			return IntentErrorResponse(fmt.Sprintf("I'm sorry but I couldn't find  %v", firstName))
		}

		rcRespBody, err := io.ReadAll(resp.Body)

		log.WithFields(log.Fields{
			"type": "rc.response.status_code"}).Info(fmt.Sprintf("%v", resp.StatusCode))
		if err != nil || resp.StatusCode >= 400 {
			log.WithFields(log.Fields{
				"type":   "rc.response",
				"status": "json.encode.error",
				"error":  fmt.Sprintf("%v", err)}).
				Warn(
					fmt.Sprintf("Error: %v\n", string(rcRespBody)))
			return IntentErrorResponse(fmt.Sprintf("An error occurred calling %v", contact.FullName()))
		} else {
			log.WithFields(log.Fields{
				"type":   "rc.response",
				"status": "response.body"}).
				Debug(fmt.Sprintf("Body: %v\n", string(rcRespBody)))
		}
	} else {
		return IntentErrorResponse(fmt.Sprintf("I couldn't find a number for %v", contact.FullName()))
	}

	actionText := fmt.Sprintf("Calling %s", contact.FullName())

	// echoResp := alexa.NewEchoResponse()
	echoResp := alexa.NewEchoResponse().OutputSpeech(
		actionText).Card("Alexa RingCentral", actionText).EndSession(true)

	return echoResp
}

func IntentErrorResponse(s string) *alexa.EchoResponse {
	return alexa.NewEchoResponse().OutputSpeech(s).Card(
		"Alexa RingCentral", s).EndSession(true)
}
