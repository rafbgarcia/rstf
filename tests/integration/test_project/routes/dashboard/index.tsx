import { useState } from "react";
import { serverData } from "@rstf/routes/dashboard";

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
        {posts.map((post, i) => (
          <li key={i}>
            {post.title} {post.published ? "(published)" : "(draft)"}
          </li>
        ))}
      </ul>
    </div>
  );
}
