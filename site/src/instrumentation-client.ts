import posthog from "posthog-js";

const posthogToken = process.env.NEXT_PUBLIC_POSTHOG_PROJECT_TOKEN;
const posthogHost =
  process.env.NEXT_PUBLIC_POSTHOG_HOST ?? "https://f.gx.run";

if (posthogToken) {
  posthog.init(posthogToken, {
    api_host: posthogHost,
    capture_pageview: "history_change",
    defaults: "2026-05-30",
  });
}
