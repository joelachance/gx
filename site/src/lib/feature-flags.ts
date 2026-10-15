/**
 * Reviews kill switch — the /reviews surface (pages, API proxies, nav
 * entries) stays in the codebase but is hidden in production while it is
 * still being worked on.
 *
 * Defaults: enabled in development, disabled in production builds. Override
 * either way with NEXT_PUBLIC_REVIEWS_ENABLED=1|0 in the deployment's build
 * environment. The flag is read by client components, so it must be
 * NEXT_PUBLIC_ and is inlined at build time — flipping it requires a
 * rebuild/redeploy, not just a restart.
 *
 * This only hides the console UI. The server's /v1/reviews endpoints and the
 * push → PR-summary pipeline are untouched, so reviews keep accumulating and
 * everything reappears when the flag turns back on.
 */
export function parseReviewsEnabled(
  raw: string | undefined,
  nodeEnv: string | undefined,
): boolean {
  const value = raw?.trim().toLowerCase();
  if (value === "1" || value === "true") return true;
  if (value === "0" || value === "false") return false;
  return nodeEnv !== "production";
}

// Literal `process.env.*` member expressions so Next.js inlines both values
// into client bundles; don't refactor these into a passed-in env object.
export const REVIEWS_ENABLED = parseReviewsEnabled(
  process.env.NEXT_PUBLIC_REVIEWS_ENABLED,
  process.env.NODE_ENV,
);
