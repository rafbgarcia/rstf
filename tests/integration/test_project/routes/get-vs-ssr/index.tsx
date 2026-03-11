import { SSR, type RoutesGetVsSsrSSRProps } from "@rstf/routes/get-vs-ssr";
import { UserAvatar } from "../../shared/ui/user-avatar";
import { useState } from "react";

export const View = SSR(function View({ message, posts }: RoutesGetVsSsrSSRProps) {
  const [count, setCount] = useState(0);
  return (
    <div>
      <title>{`Dashboard - ${message}`}</title>
      <h2 className="text-blue-500">{message}</h2>
      <UserAvatar />
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
});
