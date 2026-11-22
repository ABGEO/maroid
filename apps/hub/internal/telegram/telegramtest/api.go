// Package telegramtest gives a fake of the Telegram Bot API, so a test drives a real
// telego bot and reads each call that it made.
package telegramtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
)

// Token passes the format check of telego. No call leaves the process.
const Token = "123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw0"

const firstMessageID = 100

// Call is one method that the bot called, with its decoded parameters.
type Call struct {
	Method string
	Params map[string]any
}

// ChatID answers the chat of the call, or zero when it names none.
func (c Call) ChatID() int64 {
	if number, ok := c.Params["chat_id"].(float64); ok {
		return int64(number)
	}

	return 0
}

// Text answers the text parameter of the call.
func (c Call) Text() string {
	text, _ := c.Params["text"].(string)

	return text
}

// Markup answers the reply_markup parameter of the call, or nil.
func (c Call) Markup() map[string]any {
	markup, _ := c.Params["reply_markup"].(map[string]any)

	return markup
}

// API is the fake. It answers each message with an identifier that grows by one, and
// answers the refusal that a test set for a chat.
type API struct {
	mu        sync.Mutex
	calls     []Call
	nextID    int
	refusals  map[int64]*ta.Error
	failEdits *ta.Error
}

var _ ta.Caller = (*API)(nil)

// NewBot answers a bot whose every call reaches a new fake.
func NewBot(tb testing.TB) (*telego.Bot, *API) {
	tb.Helper()

	api := &API{nextID: firstMessageID, refusals: map[int64]*ta.Error{}}

	bot, err := telego.NewBot(Token, telego.WithAPICaller(api), telego.WithDiscardLogger())
	if err != nil {
		tb.Fatalf("creating the fake bot: %v", err)
	}

	return bot, api
}

// Refuse makes every later call to the chat answer the error of Telegram.
func (a *API) Refuse(chatID int64, code int, description string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.refusals[chatID] = &ta.Error{ErrorCode: code, Description: description}
}

// RefuseEdits makes every later edit answer the error of Telegram.
func (a *API) RefuseEdits(code int, description string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.failEdits = &ta.Error{ErrorCode: code, Description: description}
}

// Calls answers every call so far, in order.
func (a *API) Calls() []Call {
	a.mu.Lock()
	defer a.mu.Unlock()

	return append([]Call(nil), a.calls...)
}

// CallsOf answers the calls of one method, in order.
func (a *API) CallsOf(method string) []Call {
	var matching []Call

	for _, call := range a.Calls() {
		if call.Method == method {
			matching = append(matching, call)
		}
	}

	return matching
}

// Call records the call and answers it.
func (a *API) Call(_ context.Context, url string, data *ta.RequestData) (*ta.Response, error) {
	params, err := decode(data)
	if err != nil {
		return nil, err
	}

	call := Call{Method: url[strings.LastIndex(url, "/")+1:], Params: params}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.calls = append(a.calls, call)

	if refusal, refused := a.refusals[call.ChatID()]; refused {
		return &ta.Response{Ok: false, Error: refusal}, nil
	}

	if call.Method == "editMessageText" && a.failEdits != nil {
		return &ta.Response{Ok: false, Error: a.failEdits}, nil
	}

	return a.answer(call)
}

func (a *API) answer(call Call) (*ta.Response, error) {
	var result any = true

	switch call.Method {
	case "sendMessage":
		a.nextID++
		result = message(a.nextID, call)
	case "editMessageText":
		id, _ := call.Params["message_id"].(float64)
		result = message(int(id), call)
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("encoding the fake result: %w", err)
	}

	return &ta.Response{Ok: true, Result: raw}, nil
}

func message(id int, call Call) telego.Message {
	return telego.Message{
		MessageID: id,
		Chat:      telego.Chat{ID: call.ChatID(), Type: telego.ChatTypePrivate},
		Date:      time.Now().Unix(),
		Text:      call.Text(),
	}
}

func decode(data *ta.RequestData) (map[string]any, error) {
	params := map[string]any{}

	if data == nil {
		return params, nil
	}

	body := data.BodyRaw
	if body == nil && data.BodyStream != nil {
		read, err := io.ReadAll(data.BodyStream)
		if err != nil {
			return nil, fmt.Errorf("reading the fake request: %w", err)
		}

		body = read
	}

	if len(body) == 0 {
		return params, nil
	}

	if err := json.Unmarshal(body, &params); err != nil {
		return nil, fmt.Errorf("decoding the fake request: %w", err)
	}

	return params, nil
}
