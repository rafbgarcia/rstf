import { Name, Count } from "@rstf/hello/hello";

export function View() {
  return (
    <div>
      <h1>Hello {Name}</h1>
      <p>Count: {Count}</p>
    </div>
  );
}
