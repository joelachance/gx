import { source } from "@/lib/source";
import { DocsLayout } from "fumadocs-ui/layouts/docs";
import { baseOptions } from "@/lib/layout.shared";
import { DocsSearchProvider } from "@/components/docs-search-provider";

export default function Layout({ children }: LayoutProps<"/docs">) {
  return (
    <DocsSearchProvider>
      <DocsLayout tree={source.getPageTree()} {...baseOptions()}>
        {children}
      </DocsLayout>
    </DocsSearchProvider>
  );
}
