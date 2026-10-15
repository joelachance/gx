import localFont from "next/font/local";

/** Berkeley Mono — the /new landing's single typeface. Shared so the canvas
 *  terminal can render in the same family as the surrounding page. */
export const berkeleyMono = localFont({
  src: "../../fonts/BerkeleyMonoVariable.otf",
  weight: "100 900",
  display: "swap",
});
