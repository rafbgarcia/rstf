import { serverData } from "@rstf/main";
import type { ReactNode } from "react";

export function View({ children }: { children: ReactNode }) {
  const { appName } = serverData();
  return (
    <html>
      <head>
        <title>{appName}</title>
      </head>
      <body>
        <header>
          <h1>{appName}</h1>
          <nav>
            <a href="/dashboard">Dashboard</a>
          </nav>
        </header>
        <main>{children}</main>
      </body>
    </html>
  );
}
