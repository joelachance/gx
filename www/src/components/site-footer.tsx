"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { DiscordIcon } from "@/components/discord-icon";
import { GxLogo } from "@/components/gx-logo";
import { GitHubIcon } from "@/components/github-icon";
import { SatoriLogo } from "@/components/satori-logo";
import {
  DISCORD_URL,
  GITHUB_REPO_URL,
  NAV_LINKS,
  SUPPORT_EMAIL_URL,
} from "@/lib/site-links";

const linkClassName =
  "text-xs leading-5 text-zinc-600 transition-colors hover:text-zinc-950 dark:text-zinc-600 dark:hover:text-zinc-950";

const sectionLabelClassName =
  "text-xs font-medium uppercase tracking-[0.08em] text-zinc-500";

export function SiteFooter() {
  const pathname = usePathname();

  // The landing carries its own branding and links.
  if (pathname === "/") {
    return null;
  }

  return (
    <footer className="mt-auto border-t border-zinc-200 bg-white text-zinc-950 dark:border-zinc-800 dark:bg-white dark:text-zinc-950">
      <div className="mx-auto max-w-6xl px-5 pt-10 pb-8 sm:px-8 sm:pt-12">
        <div className="flex flex-col gap-10 lg:flex-row lg:items-start lg:justify-between lg:gap-16">
          <div className="flex w-fit max-w-xs flex-col items-start gap-4 text-left">
            <p className="max-w-[16rem] text-sm leading-6 text-zinc-600">
              Spend less time reviewing code
            </p>
            <GxLogo variant="footer" />
          </div>

          <nav
            aria-label="Footer"
            className="grid grid-cols-2 gap-x-10 gap-y-8 sm:gap-x-16"
          >
            <div className="space-y-8">
              <div className="space-y-3">
                <p className={sectionLabelClassName}>Product</p>
                <ul className="space-y-2">
                  {NAV_LINKS.map((link) => (
                    <li key={link.href}>
                      {link.external ? (
                        <a href={link.href} className={linkClassName}>
                          {link.label}
                        </a>
                      ) : (
                        <Link href={link.href} className={linkClassName}>
                          {link.label}
                        </Link>
                      )}
                    </li>
                  ))}
                </ul>
              </div>

              <div className="space-y-3">
                <p className={sectionLabelClassName}>Community</p>
                <ul className="space-y-2">
                  <li>
                    <a
                      href={GITHUB_REPO_URL}
                      target="_blank"
                      rel="noreferrer"
                      className={`inline-flex items-center gap-2 ${linkClassName}`}
                    >
                      <GitHubIcon className="h-3.5 w-3.5 shrink-0" />
                      GitHub
                    </a>
                  </li>
                  <li>
                    <a
                      href={DISCORD_URL}
                      target="_blank"
                      rel="noreferrer"
                      className={`inline-flex items-center gap-2 ${linkClassName}`}
                    >
                      <DiscordIcon className="h-3.5 w-3.5 shrink-0" />
                      Discord
                    </a>
                  </li>
                </ul>
              </div>
            </div>

            <div className="space-y-8">
              <div className="space-y-3">
                <p className={sectionLabelClassName}>Company</p>
                <ul className="space-y-2">
                  <li>
                    <a
                      href="https://satori.sh"
                      target="_blank"
                      rel="noreferrer"
                      className={linkClassName}
                    >
                      Website
                    </a>
                  </li>
                  <li>
                    <a href={SUPPORT_EMAIL_URL} className={linkClassName}>
                      Say hi, email us
                    </a>
                  </li>
                </ul>
              </div>

              <div className="space-y-3">
                <p className={sectionLabelClassName}>Legal</p>
                <ul className="space-y-2">
                  <li>
                    <Link href="/terms" className={linkClassName}>
                      Terms of Service
                    </Link>
                  </li>
                  <li>
                    <Link href="/privacy" className={linkClassName}>
                      Privacy Policy
                    </Link>
                  </li>
                </ul>
              </div>
            </div>
          </nav>
        </div>

        <div className="mt-10 flex items-center gap-2.5 border-t border-zinc-200 pt-5">
          <p className="text-xs font-extralight text-zinc-950">© 2026</p>
          <SatoriLogo fill="#171717" className="shrink-0" />
        </div>
      </div>
    </footer>
  );
}
