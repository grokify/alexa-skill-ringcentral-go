package voicemailcount

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/grokify/gotilla/net/httputil"
	"github.com/grokify/gotilla/time/timeutil"
	alexa "github.com/mikeflynn/go-alexa/skillserver"
	log "github.com/sirupsen/logrus"

	rchttp "github.com/grokify/ringcentral-sdk-go/rcsdk/http"

	"github.com/grokify/alexa-skill-ringcentral-go/src/config"
)

func HandleRequest(cfg config.Configuration, echoReq *alexa.EchoRequest) *alexa.EchoResponse {
	log.WithFields(log.Fields{
		"type":   "intent.voicemail",
		"status": "start handling request",
	}).Debug("Starting voicemail count request.")

	resp, err := cfg.Platform.APICall(BuildSDKRequest())

	log.WithFields(log.Fields{
		"type":   "rcsdk.status",
		"status": "voicemail status",
	}).Debug(fmt.Sprintf("Status: [%v]", resp.StatusCode))

	if err != nil {
		return IntentErrorResponse()
	}
	body, err := httputil.ResponseBody(resp)
	if err != nil {
		return IntentErrorResponse()
	}
	data := Response{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return IntentErrorResponse()
	}

	vmMessage := UnreadMessageText(data.Paging.TotalElements)
	vmMessageEcho := fmt.Sprintf("%v on RingCentral.", vmMessage)
	vmMessageCard := fmt.Sprintf("%v.", vmMessage)

	return alexa.NewEchoResponse().OutputSpeech(
		vmMessageEcho).Card("Alexa RingCentral", vmMessageCard).EndSession(true)
}

func IntentErrorResponse() *alexa.EchoResponse {
	return alexa.NewEchoResponse().OutputSpeech(
		"I could not retrieve your voicemail count").Card(
		"Alexa RingCentral", "could not retrieve voicemail count").EndSession(true)
}

func BuildSDKRequest() rchttp.Request2 {
	params := url.Values{}
	params.Add("direction", "Inbound")
	params.Add("messageType", "VoiceMail")
	params.Add("readStatus", "Unread")
	params.Add("perPage", "1000")

	dtYear, err := timeutil.NowDeltaParseDuration("-1y")
	if err == nil {
		log.WithFields(log.Fields{
			"type":   "intent.voicemail",
			"status": "got query date",
		}).Debug(fmt.Sprintf("Date: [%v]", dtYear.Format(time.RFC3339)))
		params.Add("dateFrom", dtYear.Format(time.RFC3339))
	}

	rcReq := rchttp.Request2{
		Method:  "get",
		URL:     "/account/~/extension/~/message-store",
		Query:   params,
		Headers: http.Header{}}
	rcReq.Headers.Add("Content-Type", "application/json")
	return rcReq
}

type Response struct {
	Records []interface{} `json:"records,omitempty"`
	Paging  Paging        `json:"paging,omitempty"`
}

type Paging struct {
	Page          int `json:"page,omitempty"`
	PerPage       int `json:"perPage,omitempty"`
	PageStart     int `json:"pageStart,omitempty"`
	PageEnd       int `json:"pageEnd,omitempty"`
	TotalPages    int `json:"totalPages,omitempty"`
	TotalElements int `json:"totalElements,omitempty"`
}

func UnreadMessageText(vmCount int) string {
	switch vmCount {
	case 0:
		return "You have no unread messages"
	case 1:
		return "You have 1 unread message"
	default:
		return fmt.Sprintf("You have %v unread messages", vmCount)
	}
}
