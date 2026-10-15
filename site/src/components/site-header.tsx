"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { Menu } from "lucide-react";
import { DiscordIcon } from "@/components/discord-icon";
import { GitHubIcon } from "@/components/github-icon";
import { GxWordmark } from "@/components/gx-wordmark";
import { NavSeparator } from "@/components/nav-separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { DISCORD_URL, GITHUB_REPO_URL, NAV_LINKS, type NavLink } from "@/lib/site-links";
import { cn } from "@/lib/utils";

const navLinkClassName =
  "text-sm text-zinc-600 transition-colors hover:text-zinc-950 dark:text-zinc-400 dark:hover:text-zinc-100";

const iconLinkClassName =
  "inline-flex h-8 w-8 items-center justify-center text-zinc-500 transition-colors hover:text-zinc-950 dark:text-zinc-400 dark:hover:text-zinc-100";

function SocialIconLinks() {
  return (
    <>
      <a
        href={GITHUB_REPO_URL}
        target="_blank"
        rel="noreferrer"
        className={iconLinkClassName}
        aria-label="GitHub repository"
      >
        <GitHubIcon className="h-[18px] w-[18px]" />
      </a>
      <a
        href={DISCORD_URL}
        target="_blank"
        rel="noreferrer"
        className={iconLinkClassName}
        aria-label="Discord community"
      >
        <DiscordIcon className="h-[18px] w-[18px]" />
      </a>
    </>
  );
}

function SocialIconCluster({ className }: { className?: string }) {
  return (
    <div className={cn("flex items-center gap-2", className)}>
      <NavSeparator />
      <div className="flex items-center gap-1 px-3 sm:px-4">
        <SocialIconLinks />
      </div>
      <NavSeparator />
    </div>
  );
}

const HEADER_SCROLL_OFFSET = 60;

function scrollToHash(hash: string) {
  const id = hash.replace(/^#/, "");
  const target = document.getElementById(id);
  if (!target) return false;

  const top =
    target.getBoundingClientRect().top + window.scrollY - HEADER_SCROLL_OFFSET;

  window.scrollTo({ top: Math.max(0, top), behavior: "smooth" });
  window.history.pushState(null, "", hash);
  return true;
}

function scrollToHashAfterUnlock(hash: string) {
  window.setTimeout(() => {
    scrollToHash(hash);
  }, 300);
}

function NavAnchorLink({
  href,
  label,
  className,
  onNavigate,
  deferScroll,
}: {
  href: string;
  label: string;
  className?: string;
  onNavigate?: () => void;
  deferScroll?: boolean;
}) {
  const pathname = usePathname();
  const hashIndex = href.indexOf("#");
  const hash = hashIndex >= 0 ? href.slice(hashIndex) : null;
  const path = hash ? href.slice(0, hashIndex) || "/" : href;

  return (
    <Link
      href={href}
      className={cn(navLinkClassName, className)}
      onClick={(event) => {
        if (hash && pathname === path) {
          event.preventDefault();

          if (deferScroll) {
            onNavigate?.();
            return;
          }

          scrollToHash(hash);
          onNavigate?.();
          return;
        }

        onNavigate?.();
      }}
    >
      {label}
    </Link>
  );
}

function HeaderNavLink({
  link,
  className,
  onNavigate,
  deferScroll,
}: {
  link: NavLink;
  className?: string;
  onNavigate?: () => void;
  deferScroll?: boolean;
}) {
  if (link.external) {
    return (
      <a
        href={link.href}
        className={cn(navLinkClassName, className)}
        onClick={onNavigate}
      >
        {link.label}
      </a>
    );
  }

  return (
    <NavAnchorLink
      href={link.href}
      label={link.label}
      className={className}
      deferScroll={deferScroll}
      onNavigate={onNavigate}
    />
  );
}

function MobileNavMenu() {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const pendingHashRef = useRef<string | null>(null);

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);

    if (!nextOpen && pendingHashRef.current) {
      const hash = pendingHashRef.current;
      pendingHashRef.current = null;
      scrollToHashAfterUnlock(hash);
    }
  };

  const queueSectionScroll = (href: string) => {
    const hashIndex = href.indexOf("#");
    const hash = hashIndex >= 0 ? href.slice(hashIndex) : null;
    const path = hash ? href.slice(0, hashIndex) || "/" : href;

    if (hash && pathname === path) {
      pendingHashRef.current = hash;
    }
  };

  const closeMenu = (href: string) => {
    queueSectionScroll(href);
    handleOpenChange(false);
  };

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetTrigger asChild>
        <button
          type="button"
          className="inline-flex h-9 w-9 items-center justify-center text-zinc-600 transition-colors hover:text-zinc-950 dark:text-zinc-400 dark:hover:text-zinc-100 lg:hidden"
          aria-label="Open menu"
        >
          <Menu className="h-5 w-5" />
        </button>
      </SheetTrigger>
      <SheetContent className="flex w-full max-w-xs flex-col border-zinc-200 p-0 dark:border-zinc-800 sm:max-w-xs">
        <SheetHeader>
          <SheetTitle className="text-base font-medium">Menu</SheetTitle>
        </SheetHeader>
        <nav
          aria-label="Mobile primary"
          className="flex flex-col gap-1 px-6 pb-6"
        >
          {NAV_LINKS.map((link) => (
            <HeaderNavLink
              key={link.href}
              link={link}
              className="py-2.5 text-base"
              deferScroll
              onNavigate={() => closeMenu(link.href)}
            />
          ))}
        </nav>
        <div className="mt-auto border-t border-zinc-200 px-6 py-6 dark:border-zinc-800">
          <div className="flex items-center gap-1">
            <SocialIconLinks />
          </div>
          <div className="mt-6 space-y-4">
            <a
              href="https://github.com/satoricorp/gx"
              className="block text-sm text-zinc-600 transition-colors hover:text-zinc-950 dark:text-zinc-400 dark:hover:text-zinc-100"
            >
              Source on GitHub
            </a>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}

export function SiteHeader() {
  const pathname = usePathname();
  const isHome = pathname === "/";
  const [belowFold, setBelowFold] = useState(false);

  useEffect(() => {
    let frame: number | null = null;

    const updateBelowFold = () => {
      frame = null;
      setBelowFold(window.scrollY >= window.innerHeight);
    };

    const scheduleUpdate = () => {
      if (frame !== null) {
        return;
      }

      frame = requestAnimationFrame(updateBelowFold);
    };

    if (!isHome) {
      const resetFrame = requestAnimationFrame(() => setBelowFold(false));

      return () => cancelAnimationFrame(resetFrame);
    }

    scheduleUpdate();
    window.addEventListener("scroll", scheduleUpdate, { passive: true });
    window.addEventListener("resize", scheduleUpdate);

    return () => {
      if (frame !== null) {
        cancelAnimationFrame(frame);
      }

      window.removeEventListener("scroll", scheduleUpdate);
      window.removeEventListener("resize", scheduleUpdate);
    };
  }, [isHome]);

  useEffect(() => {
    if (pathname !== "/" || !window.location.hash) return;

    const hash = window.location.hash;
    requestAnimationFrame(() => {
      scrollToHash(hash);
    });
  }, [pathname]);

  const showNavLogo = !isHome || belowFold;

  // The landing carries its own branding and links.
  if (isHome) {
    return null;
  }

  return (
    <header className="sticky top-0 z-50 border-b border-zinc-200 bg-background/95 py-3 backdrop-blur-sm dark:border-zinc-800">
      <div className="relative flex w-full items-center gap-2 pl-0.5 pr-5 sm:pr-10">
        <Link
          href="/"
          aria-label="gx home"
          aria-hidden={!showNavLogo}
          tabIndex={showNavLogo ? undefined : -1}
          className={cn(
            "flex shrink-0 items-center overflow-hidden transition-[max-width,opacity] duration-500 ease-out",
            showNavLogo
              ? "max-w-[5rem] opacity-100"
              : "pointer-events-none max-w-0 opacity-0",
          )}
        >
          <GxWordmark size="nav" className="px-3 sm:px-4" />
        </Link>

        <nav
          aria-label="Primary"
          className="absolute left-1/2 hidden -translate-x-1/2 items-center gap-8 lg:flex"
        >
          {NAV_LINKS.map((link) => (
            <HeaderNavLink key={link.href} link={link} />
          ))}
        </nav>

        <div className="ml-auto flex shrink-0 items-center gap-3 sm:gap-4">
          <SocialIconCluster className="hidden xl:flex" />
          {/*
            This held "Sign in" and "Get started", which resolved to a session
            check and an OAuth route. Neither exists now, so the right-hand
            cluster is the source link — the same destination the mobile sheet
            and the landing page use.
          */}
          <a
            href={GITHUB_REPO_URL}
            className={`${navLinkClassName} hidden sm:inline`}
          >
            Source
          </a>
          <MobileNavMenu />
        </div>
      </div>
    </header>
  );
}
