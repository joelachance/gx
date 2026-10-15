import { readDocsFile } from "@/lib/docs-content";

// Read at build time from docs/, never per-request, so it is a static asset
// like any other under output: "export". Without this Next refuses the build:
// a route with no revalidation is ambiguous under export, and it would rather
// guess wrong than silently emit a server route into a static bundle.
export const dynamic = "force-static";

export function GET() {
  return new Response(readDocsFile("llms.txt"), {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
}
