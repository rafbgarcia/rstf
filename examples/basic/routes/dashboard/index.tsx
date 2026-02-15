import { serverData } from "@rstf/routes/dashboard";

export function View() {
  const { message, posts } = serverData();
  return (
    <div>
      <h2>{message}</h2>
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
