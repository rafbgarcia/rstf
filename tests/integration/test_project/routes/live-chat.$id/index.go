package livechat

import (
	"strings"

	rstf "github.com/rafbgarcia/rstf"
)

type Message struct {
	Body string `json:"body"`
}

type GetMessagesResult struct {
	Messages []Message `json:"messages"`
}

type SendMessageInput struct {
	Body string `json:"body"`
}

type EchoActionResult struct {
	Value string `json:"value"`
}

var (
	liveChatMessages = map[string][]Message{
		"room-1": {
			{Body: "Hello from the server"},
		},
	}
)

func GetMessages(ctx *rstf.QueryContext) GetMessagesResult {
	roomID := ctx.Param("id")
	messages := append([]Message(nil), liveChatMessages[roomID]...)
	return GetMessagesResult{Messages: messages}
}

func SendMessage(ctx *rstf.MutationContext, input SendMessageInput) error {
	roomID := ctx.Param("id")
	liveChatMessages[roomID] = append(liveChatMessages[roomID], Message{
		Body: strings.TrimSpace(input.Body),
	})

	ctx.Invalidate(rstf.NewSubscriptionKey("live-chat.$id", "GetMessages", map[string]string{"id": roomID}))
	return nil
}

func EchoAction(ctx *rstf.ActionContext, input string) EchoActionResult {
	return EchoActionResult{Value: strings.ToUpper(input)}
}
