import type { Metadata } from "next";
import localFont from "next/font/local";
import { Geist_Mono } from "next/font/google";
import { Suspense } from "react";
import { RootProvider } from "fumadocs-ui/provider/next";
import "./globals.css";
import { ApolloTracker } from "@/components/apollo-tracker";
import { GoogleAnalytics } from "@/components/google-analytics";
import { SiteFooter } from "@/components/site-footer";
import { SiteHeader } from "@/components/site-header";

const fkDisplay = localFont({
  src: "../fonts/FKDisplay-Regular.otf",
  variable: "--font-fk-display",
  display: "swap",
});

const xer0 = localFont({
  src: "../fonts/Xer0-Regular.otf",
  variable: "--font-xer0",
  display: "swap",
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "gx: Spend less time reviewing code",
  description:
    "gx reviews your change before you push with /gx, answers questions in your pull request with @gx, and ranks every change by how much it matters. Set up once with gx init.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${fkDisplay.variable} ${xer0.variable} ${geistMono.variable} h-full antialiased`}
      suppressHydrationWarning
    >
      <body className="flex min-h-screen flex-col font-sans">
        <RootProvider
          theme={{
            attribute: "class",
            defaultTheme: "system",
            enableSystem: true,
            disableTransitionOnChange: true,
          }}
          search={{ enabled: false }}
        >
          <Suspense fallback={null}>
            <GoogleAnalytics />
          </Suspense>
          <ApolloTracker />
          <SiteHeader />
          {/*
            The gate this Suspense wrapped existed to hold a spinner while
            Convex answered whether the visitor was signed in, so the header
            could show the right links. There is no account and nothing to ask,
            so the children render directly.
          */}
          {children}
          <SiteFooter />
        </RootProvider>
      </body>
    </html>
  );
}
