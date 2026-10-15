import type { Metadata } from "next";
import { LandingV2 } from "@/components/landing-v2/landing";

export const metadata: Metadata = {
  title: "gx: Code review that knows how the code was written",
  description:
    "gx records the agent session behind each commit and reviews your changes with that context. Findings before you push, summaries and answers in your PR. Plain Git, two commands.",
};

// This used to be async and redirected signed-in visitors to the app. There is
// no app and nobody to sign in, so every visitor gets the landing page and the
// page needs no request at all — which is what makes the static export below
// possible for it.
export default function Home() {
  return <LandingV2 />;
}