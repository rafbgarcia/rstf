import { SSR, type MainSSRProps } from "@rstf/main";

export const View = SSR(function View({ children, appName }: MainSSRProps) {
  return (
    <html>
      <head>
        <title>{appName}</title>
      </head>
      <body>
        <header>
          <h1>{appName}</h1>
          <nav>
            <a href="/get-vs-ssr">Dashboard</a>
          </nav>
        </header>
        <main>{children}</main>
      </body>
    </html>
  );
});
