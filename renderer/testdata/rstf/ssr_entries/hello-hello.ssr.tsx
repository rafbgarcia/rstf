import { renderToString } from "react-dom/server.browser";
import { SSRDataProvider } from "@rstf/ssr";
import { View as Layout } from "../../layout/layout";
import { View as Route } from "../../hello/hello";

const render = (ssrProps: Record<string, Record<string, any>>) =>
  renderToString(
    <SSRDataProvider data={ssrProps}>
      <Layout>
        <Route />
      </Layout>
    </SSRDataProvider>
  );

(globalThis as any).__RSTF_RENDERERS__ = (globalThis as any).__RSTF_RENDERERS__ ?? {};
(globalThis as any).__RSTF_RENDERERS__["hello/hello"] = render;
