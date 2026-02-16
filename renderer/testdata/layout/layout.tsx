import { serverData } from "@rstf/layout/layout";
import type { ReactNode } from "react";

export function View({ children }: { children: ReactNode }) {
  const { title } = serverData();
  return (
    <html>
      <head>
        <title>{title}</title>
      </head>
      <body>
        <header>{title}</header>
        <main>{children}</main>
      </body>
    </html>
  );
}
