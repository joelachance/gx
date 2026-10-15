export const GITHUB_REPO_URL =
  process.env.NEXT_PUBLIC_GITHUB_REPO_URL ??
  "https://github.com/satoricorp/gx";

export const DISCORD_URL =
  process.env.NEXT_PUBLIC_DISCORD_URL ?? "https://discord.gg/NJ2aGugtj";

/** Company/product Twitter for now — Joe's handle. */
export const TWITTER_URL =
  process.env.NEXT_PUBLIC_TWITTER_URL ?? "https://x.com/satori_corp";
export const TWITTER_HANDLE = "@satori_corp";

export const SUPPORT_EMAIL = "joe@intramural.co";
export const SUPPORT_EMAIL_URL = `mailto:${SUPPORT_EMAIL}`;

/** Founder booking link behind the landing CTA. */
export const FOUNDER_CALL_URL =
  process.env.NEXT_PUBLIC_FOUNDER_CALL_URL ??
  "https://cal.com/meet-with-satori/15min";

/** Where "Get started" points. There is no OAuth flow any more, so this is
 * just the install page. */
export const POST_SIGN_IN_URL = "/download";
export const DOCS_URL = "/docs";

export type NavLink = {
  label: string;
  href: string;
  external?: boolean;
};

export const NAV_LINKS: NavLink[] = [
  { label: "Documentation", href: DOCS_URL },
];
