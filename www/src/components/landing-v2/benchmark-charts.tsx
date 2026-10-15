/** The two bar charts on the Proof & Benchmarks slide.
 *
 * One tool per row, value at the bar tip, so every mark names itself — no
 * legend to cross-reference. GX wears the site accent; the competitors sit on
 * a gray ramp (checked for adjacent-step separation and 3:1 contrast against
 * the page surface), so the one colored series is the one the slide is about.
 *
 * True positives and missed counts are omitted on purpose: both are recall in
 * other clothes (each divides by the same 158 real issues), so charting them
 * would show the same comparison twice.
 */

const TOOLS = [
  { name: "GX", color: "#ff80ff", self: true },
  { name: "Greptile", color: "#757575", self: false },
  { name: "CodeRabbit", color: "#A6A6A6", self: false },
  { name: "Claude Code", color: "#EDEDED", self: false },
] as const;

type Tool = (typeof TOOLS)[number];

/** Values follow TOOLS order. Percent bars share one 0–75 scale: the largest
 * value is 58.2, and the headroom keeps tip labels inside the column even on
 * phone-width tracks. */
const PERCENT_MAX = 75;
const DETECTION_METRICS = [
  { label: "Precision", values: [47.7, 34.7, 33.8, 36.5] },
  { label: "Recall", values: [46.8, 43.7, 58.2, 41.1] },
  { label: "F1 score", values: [47.3, 38.7, 42.8, 38.7] },
] as const;

const FALSE_POSITIVE_MAX = 220;
const FALSE_POSITIVES = [81, 130, 180, 113] as const;

function BarRow({
  tool,
  value,
  max,
  unit,
}: {
  tool: Tool;
  value: number;
  max: number;
  unit?: string;
}) {
  const pct = (value / max) * 100;
  return (
    <div className="flex items-center gap-2">
      <span
        className={`w-[5.5rem] shrink-0 text-right text-[10px] leading-none ${
          tool.self ? "text-zinc-100" : "text-zinc-400"
        }`}
      >
        {tool.name}
      </span>
      <div className="relative h-2.5 flex-1">
        <div
          className="h-full"
          style={{
            width: `${pct}%`,
            background: tool.color,
            // Square at the baseline, rounded at the data end.
            borderRadius: "0 4px 4px 0",
          }}
        />
        <span
          className={`absolute top-1/2 -translate-y-1/2 pl-1.5 text-[10px] leading-none ${
            tool.self ? "text-zinc-100" : "text-zinc-400"
          }`}
          style={{ left: `${pct}%` }}
        >
          {value}
          {unit}
        </span>
      </div>
    </div>
  );
}

function ChartHeader({ title, note }: { title: string; note: string }) {
  return (
    <div className="mb-2 flex items-baseline justify-between gap-3">
      <p className="text-[10px] uppercase leading-none tracking-[0.08em] text-zinc-400">
        {title}
      </p>
      <p className="text-[10px] leading-none text-zinc-400">{note}</p>
    </div>
  );
}

/** The shared zero line the bars grow from — recessive, one step off the
 * surface, spanning the rows it is placed behind. */
function Baseline() {
  return (
    <div
      aria-hidden
      className="absolute inset-y-0 w-px bg-[#33312f]"
      style={{ left: "calc(5.5rem + 8px)" }}
    />
  );
}

export function BenchmarkCharts() {
  return (
    <div className="space-y-6">
      <figure>
        <ChartHeader title="Detection quality" note="&uarr; higher is better" />
        <div className="relative space-y-3">
          <Baseline />
          {DETECTION_METRICS.map((metric) => (
            <div key={metric.label}>
              <p className="mb-1.5 pl-[calc(5.5rem+8px)] text-[10px] leading-none text-zinc-400">
                {metric.label}
              </p>
              <div className="space-y-1">
                {TOOLS.map((tool, i) => (
                  <BarRow
                    key={tool.name}
                    tool={tool}
                    value={metric.values[i]}
                    max={PERCENT_MAX}
                    unit="%"
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      </figure>

      <figure>
        <ChartHeader title="False positives" note="&darr; lower is better" />
        <div className="relative space-y-1">
          <Baseline />
          {TOOLS.map((tool, i) => (
            <BarRow
              key={tool.name}
              tool={tool}
              value={FALSE_POSITIVES[i]}
              max={FALSE_POSITIVE_MAX}
            />
          ))}
        </div>
      </figure>
    </div>
  );
}
