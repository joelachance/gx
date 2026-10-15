"use client";

import type { ReactNode } from "react";
import { SearchProvider } from "fumadocs-ui/contexts/search";

/** Docs-only Fumadocs search so ⌘K doesn't compete with the site palette. */
export function DocsSearchProvider({ children }: { children: ReactNode }) {
  return <SearchProvider>{children}</SearchProvider>;
}
