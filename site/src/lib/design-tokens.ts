/** Design reference tokens for the Console UI. */

export const SEMANTIC_COLORS = [
  { token: "--background", light: "#ffffff", dark: "#0a0a0a" },
  { token: "--foreground", light: "#171717", dark: "#ededed" },
  { token: "--ascii-muted", light: "#e8e8e8", dark: "#121212" },
  { token: "--ascii-gray", light: "#333333", dark: "#333333" },
  { token: "--footer-link-hover", light: "#a360a3", dark: "#a360a3" },
] as const;

export const UI_ZINC = [
  { token: "zinc-50", hex: "#fafafa" },
  { token: "zinc-100", hex: "#f4f4f5" },
  { token: "zinc-200", hex: "#e4e4e7" },
  { token: "zinc-300", hex: "#d4d4d8" },
  { token: "zinc-500", hex: "#71717a" },
  { token: "zinc-600", hex: "#52525b" },
  { token: "zinc-700", hex: "#3f3f46" },
  { token: "zinc-800", hex: "#27272a" },
  { token: "zinc-900", hex: "#18181b" },
  { token: "zinc-950", hex: "#09090b" },
] as const;

export const ACCENT_COLORS = [{ token: "amber-500", hex: "#f59e0b" }] as const;

export const TYPOGRAPHY = [
  {
    name: "FK Display Trial",
    variable: "--font-fk-display",
    stack: 'var(--font-fk-display), ui-sans-serif, system-ui, sans-serif',
  },
  {
    name: "Geist Mono",
    variable: "--font-geist-mono",
    stack: "var(--font-geist-mono), ui-monospace, monospace",
  },
] as const;

export const ICON_EXPORT_SIZES = [
  { label: "favicon-16", size: 16, mark: "x" },
  { label: "favicon-32", size: 32, mark: "x" },
  { label: "favicon-48", size: 48, mark: "x" },
  { label: "apple-touch-icon", size: 180, mark: "gx" },
  { label: "pwa-192", size: 192, mark: "gx" },
  { label: "pwa-512", size: 512, mark: "gx" },
  { label: "app-store", size: 1024, mark: "gx" },
] as const;

export const DESIGN_SECTIONS = [
  { id: "colors", label: "Colors" },
  { id: "typography", label: "Typography" },
  { id: "components", label: "Components" },
  { id: "icon-export", label: "Icon export" },
] as const;

export type DesignSectionId = (typeof DESIGN_SECTIONS)[number]["id"];
