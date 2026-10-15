import { cn } from "@/lib/utils";

type GxWordmarkProps = {
  className?: string;
  /** Visual size preset for nav vs hero. */
  size?: "nav" | "hero";
};

const sizeClassName = {
  nav: "text-[1.65rem] leading-none tracking-tight",
  hero: "text-5xl leading-none tracking-tight sm:text-6xl",
} as const;

export function GxWordmark({ className, size = "nav" }: GxWordmarkProps) {
  return (
    <span
      aria-hidden
      className={cn(
        "font-[family-name:var(--font-xer0)] text-zinc-950 dark:text-zinc-50",
        sizeClassName[size],
        className,
      )}
    >
      gx
    </span>
  );
}
