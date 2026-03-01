import { serverData } from "@rstf/routes/get-vs-ssr";
import { useState } from "react";

export function View() {
  const { message, posts } = serverData();
  const [count, setCount] = useState(0);
  return (
    <div>
      <title>{`Dashboard - ${message}`}</title>
      <h2 className="text-blue-500">{message}</h2>
      <button data-testid="counter" onClick={() => setCount((c) => c + 1)}>
        Count: {count}
      </button>
      <ul>
        {posts.map((post) => (
          <li key={post.title}>
            {post.title} {post.published ? "(published)" : "(draft)"}
          </li>
        ))}
      </ul>
    </div>
  );
}
