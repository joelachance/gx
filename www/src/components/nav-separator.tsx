import { cn } from "@/lib/utils";

export function NavSeparator({ className }: { className?: string }) {
  return (
    <span
      aria-hidden
      className={cn(
        "hidden h-4 w-px shrink-0 bg-zinc-200 sm:block dark:bg-zinc-800",
        className,
      )}
    />
  );
}
