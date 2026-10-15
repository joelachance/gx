const LOGO_ASPECT = 1024 / 69;
const LOGO_MASK = "url(/satori-engineering-co.png)";

type SatoriLogoProps = {
  height?: number;
  className?: string;
  fill?: string;
};

export function SatoriLogo({
  height = 12,
  className,
  fill = "#ffffff",
}: SatoriLogoProps) {
  const width = Math.round(LOGO_ASPECT * height);

  return (
    <span
      role="img"
      aria-label="Satori Engineering Co."
      className={className}
      style={{
        display: "inline-block",
        width,
        height,
        backgroundColor: fill,
        WebkitMaskImage: LOGO_MASK,
        WebkitMaskSize: "100% 100%",
        WebkitMaskRepeat: "no-repeat",
        maskImage: LOGO_MASK,
        maskSize: "100% 100%",
        maskRepeat: "no-repeat",
      }}
    />
  );
}
