# Live Queries

`rstf` now has a route-derived RPC model for server-owned live state.

The core idea is:

- `useQuery` reads server-owned state and stays live
- `useMutation` changes server-owned state
- `useAction` runs side-effectful or non-deterministic work

## Server Function Kinds

In a route Go file, use these context types:

- `*rstf.QueryContext`
- `*rstf.MutationContext`
- `*rstf.ActionContext`

Example:

```go
package livechat

import (
	"strings"

	rstf "github.com/rafbgarcia/rstf"
	"example.com/my-app/rstf/routes"
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

func GetMessages(ctx *rstf.QueryContext) GetMessagesResult {
	roomID := ctx.Param("id")
	return GetMessagesResult{
		Messages: loadMessages(roomID),
	}
}

func SendMessage(ctx *rstf.MutationContext, input SendMessageInput) error {
	roomID := ctx.Param("id")
	saveMessage(roomID, strings.TrimSpace(input.Body))

	routes.LiveChatDotIdGetMessages.Invalidate(
		ctx,
		routes.LiveChatDotIdParams{Id: roomID},
	)
	return nil
}
```

## Client Hooks

The generated route module exports typed descriptors and hooks:

```tsx
import { routes, useAction, useMutation, useQuery } from "@rstf/routes";

export function View() {
  const query = useQuery(routes["live-chat._id"].GetMessages, { id: "room-1" });
  const sendMessage = useMutation(routes["live-chat._id"].SendMessage, { id: "room-1" });
  const echo = useAction(routes["live-chat._id"].EchoAction, { id: "room-1" });

  if (query.status === "loading") {
    return <div>Loading...</div>;
  }

  if (query.status === "error") {
    return <div>{query.error.message}</div>;
  }

  return (
    <div>
      <ul>
        {query.data.messages.map((message, index) => (
          <li key={index}>{message.body}</li>
        ))}
      </ul>
      <button onClick={() => sendMessage({ body: "Hello" })}>Send</button>
      <button onClick={() => echo("hello")}>Echo</button>
    </div>
  );
}
```

`useMutation` and `useAction` both return a function. The hook binds route params once, and each call passes that function's input payload.

## Query Semantics

Today, `useQuery` is always live.

That means:

- the client subscribes over a shared SSE connection
- the server keeps the query subscription in memory
- a mutation invalidates one or more generated query keys
- the server reruns the matching query
- the client receives a fresh full snapshot

The pushed payload is a full query result, not a patch.

## Mutation vs Action

`useMutation` and `useAction` look similar on the client, but they mean different things.

Use `Mutation` for:

- deterministic app-state writes
- invalidating live queries
- framework-managed state changes

Use `Action` for:

- external HTTP calls
- email
- webhooks
- other non-deterministic side effects

## Type-Safe Invalidation

Do not build invalidation keys by hand.

The generated Go route package exposes typed query helpers:

```go
routes.LiveChatDotIdGetMessages.Invalidate(
	ctx,
	routes.LiveChatDotIdParams{Id: roomID},
)
```

That keeps invalidation derived from the actual route contract instead of raw strings.

## Current Runtime Shape

The current live runtime is:

- single-instance in-memory subscription storage
- one SSE connection per browser tab
- multiplexed live query events over that SSE stream
- full-snapshot query updates

Distributed backplanes are not documented here yet because the current user-facing workflow is still focused on the single-instance local-to-dev loop.
