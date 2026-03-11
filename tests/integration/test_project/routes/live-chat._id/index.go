package livechat

import (
	"strings"

	rstf "github.com/rafbgarcia/rstf"
	"github.com/rafbgarcia/rstf/tests/integration/test_project/rstf/routes"
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

	routes.LiveChatDotIdGetMessages.Invalidate(ctx, routes.LiveChatDotIdParams{Id: roomID})
	return nil
}

func EchoAction(ctx *rstf.ActionContext, input string) EchoActionResult {
	return EchoActionResult{Value: strings.ToUpper(input)}
}
