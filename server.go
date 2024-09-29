package rcskillserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/buaazp/fasthttprouter"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"

	"github.com/grokify/mogo/net/http/httputilmore"
	alexa "github.com/mikeflynn/go-alexa/skillserver"

	"github.com/grokify/alexa-skill-ringcentral-go/src/config"
	"github.com/grokify/alexa-skill-ringcentral-go/src/intents/ringout"
	"github.com/grokify/alexa-skill-ringcentral-go/src/intents/sms"
	smsbody "github.com/grokify/alexa-skill-ringcentral-go/src/intents/sms_body"
	voicemailcount "github.com/grokify/alexa-skill-ringcentral-go/src/intents/voicemail_count"
)

const (
	RouteRingCentral = "/echo/ringcentral"
)

type Handler struct {
	Configuration config.Configuration
}

func NewHandler(cfg config.Configuration) Handler {
	cfg.AddressBook.Inflate()
	return Handler{Configuration: cfg}
}

func (h *Handler) HandleFastHTTP(fctx *fasthttp.RequestCtx) {
	echoReq := &alexa.EchoRequest{}
	err := json.Unmarshal(fctx.PostBody(), echoReq)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error()})
		fctx.SetContentType(httputilmore.ContentTypeTextPlainUsASCII)
		fctx.SetStatusCode(http.StatusInternalServerError)
		fctx.SetBodyString("Internal Server Error")
	}

	log.WithFields(log.Fields{
		"sessionID": echoReq.Session.SessionID,
		"body":      fmt.Sprintf("%v", fctx.PostBody())}).Info("HandleFastHTTP Request")

	var echoResp *alexa.EchoResponse

	ctx := context.Background()

	switch echoReq.GetIntentName() {
	case "RingCentralGetNewVoicemailCountIntent":
		echoResp = voicemailcount.HandleRequest(ctx, h.Configuration, echoReq)
	case "RingCentralSendSMSIntent":
		echoResp = sms.HandleRequest(ctx, h.Configuration, echoReq)
	case "RingCentralSendSMSIntentBody":
		echoResp = smsbody.HandleRequest(ctx, h.Configuration, echoReq)
	case "RingCentralCreateRingOutIntent":
		echoResp = ringout.HandleRequest(ctx, h.Configuration, echoReq)
	default:
		echoResp = alexa.NewEchoResponse().OutputSpeech("I'm sorry, I didn't get that. Can you say that again?").EndSession(false)
	}

	echoRespBytes, err := json.Marshal(echoResp)
	if err != nil {
		log.WithFields(log.Fields{
			"sessionID": echoReq.Session.SessionID,
			"body":      fmt.Sprintf("%v", fctx.PostBody())}).Warn("HandleFastHTTP Response JSON Marshal Error")
	}
	log.WithFields(log.Fields{
		"sessionID": echoReq.Session.SessionID,
		"body":      string(echoRespBytes)}).Info("HandleFastHTTP Response")

	fctx.Response.Header.Set(httputilmore.HeaderContentType, httputilmore.ContentTypeAppJSONUtf8)
	fmt.Fprintln(fctx, string(echoRespBytes))
}

// StartServer initializes and starts the webhook proxy server
func StartServer(cfg config.Configuration) {
	log.SetLevel(log.InfoLevel)

	router := fasthttprouter.New()

	rcHandler := NewHandler(cfg)
	router.POST(RouteRingCentral, rcHandler.HandleFastHTTP)

	log.WithFields(log.Fields{}).Info(fmt.Sprintf("Listening on port %v.", cfg.Port))
	log.Fatal(fasthttp.ListenAndServe(cfg.Address(), router.Handler))
}
