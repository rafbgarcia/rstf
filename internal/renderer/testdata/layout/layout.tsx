import { Title } from "@rstf/layout/layout";
import type { ReactNode } from "react";

export function View({ children }: { children: ReactNode }) {
  return (
    <html>
      <head>
        <title>{Title}</title>
      </head>
      <body>
        <header>{Title}</header>
        <main>{children}</main>
      </body>
    </html>
  );
}
