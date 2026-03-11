import { SSR, type HelloHelloSSRProps } from "@rstf/hello/hello";

export const View = SSR(function View({ name, count }: HelloHelloSSRProps) {
  return (
    <div>
      <h1>Hello {name}</h1>
      <p>Count: {count}</p>
    </div>
  );
});
