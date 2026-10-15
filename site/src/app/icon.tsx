import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { ImageResponse } from "next/og";

// Generated once at build time; a static export has no request to vary by.
export const dynamic = "force-static";

export const size = {
  width: 32,
  height: 32,
};

export const contentType = "image/png";

const blob = await readFile(
  join(process.cwd(), "public/fonts/Blob-Regular.ttf"),
);

export default function Icon() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: "#0a0a0a",
          color: "#fafafa",
          fontFamily: "Blob",
          fontSize: 26,
          lineHeight: 1,
        }}
      >
        X
      </div>
    ),
    {
      ...size,
      fonts: [
        {
          name: "Blob",
          data: blob,
          style: "normal",
          weight: 400,
        },
      ],
    },
  );
}
