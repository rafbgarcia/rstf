import { useState } from "react";
import { routes, useMutation, useQuery } from "@rstf/routes";

export function View() {
  const [draft, setDraft] = useState("");
  const query = useQuery(routes["live-chat.$id"].GetMessages, { id: "room-1" });
  const sendMessage = useMutation(routes["live-chat.$id"].SendMessage, { id: "room-1" });

  if (query.status === "loading") {
    return <div data-testid="messages-loading">Loading messages...</div>;
  }

  if (query.status === "error") {
    return <div data-testid="messages-error">{query.error.message}</div>;
  }

  return (
    <div>
      <h2>Live Chat</h2>
      <ul data-testid="messages-list">
        {query.data.messages.map((message, index) => (
          <li key={`${message.body}-${index}`}>{message.body}</li>
        ))}
      </ul>
      <input
        data-testid="chat-input"
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
      />
      <button
        data-testid="send-message"
        onClick={async () => {
          await sendMessage({ body: draft });
          setDraft("");
        }}
      >
        Send
      </button>
    </div>
  );
}
