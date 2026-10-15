import defaultMdxComponents from "fumadocs-ui/mdx";
import { Callout } from "fumadocs-ui/components/callout";
import type { MDXComponents } from "mdx/types";
import type { ReactNode } from "react";

function Note({ children }: { children: ReactNode }) {
  return <Callout type="info">{children}</Callout>;
}

export function getMDXComponents(components?: MDXComponents): MDXComponents {
  return {
    ...defaultMdxComponents,
    Note,
    ...components,
  } as MDXComponents;
}
