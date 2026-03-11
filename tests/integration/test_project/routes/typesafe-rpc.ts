import { routes, useAction, useMutation, useQuery } from "@rstf/routes";

function Example() {
  const query = useQuery(routes["live-chat._id"].GetMessages, { id: "room-1" });
  const sendMessage = useMutation(routes["live-chat._id"].SendMessage, { id: "room-1" });
  const echo = useAction(routes["live-chat._id"].EchoAction, { id: "room-1" });

  query.data;
  void sendMessage({ body: "hello" });
  void echo("hello");

  // @ts-expect-error queries cannot be passed to useMutation
  useMutation(routes["live-chat._id"].GetMessages, { id: "room-1" });
  // @ts-expect-error mutations cannot be passed to useQuery
  useQuery(routes["live-chat._id"].SendMessage, { id: "room-1" });
  // @ts-expect-error actions require useAction
  useMutation(routes["live-chat._id"].EchoAction, { id: "room-1" });

  return query.status;
}

void Example;
