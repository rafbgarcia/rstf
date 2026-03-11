import { SSR, type LayoutLayoutSSRProps } from "@rstf/layout/layout";

export const View = SSR(function View({ children, title }: LayoutLayoutSSRProps) {
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
});
