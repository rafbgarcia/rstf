import { serverData } from "@rstf/hello/hello";

export function View() {
  const { name, count } = serverData();
  return (
    <div>
      <h1>Hello {name}</h1>
      <p>Count: {count}</p>
    </div>
  );
}
