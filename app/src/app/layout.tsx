import type { Metadata } from "next";
import "./globals.css";
import { Toaster } from "@/components/ui/sonner";
export const metadata: Metadata = {
  metadataBase: new URL("https://sentinel.acme.example"),
  title: "Sentinel",
  description:
    "AI governance console: observe model traffic, enforce policy, and prove compliance across every project.",
  applicationName: "Sentinel",
  openGraph: {
    type: "website",
    siteName: "Sentinel",
    title: "Sentinel — AI governance console",
    description:
      "Observe model traffic, enforce policy, and prove compliance across every project.",
    url: "/",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "Sentinel" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "Sentinel — AI governance console",
    description:
      "Observe model traffic, enforce policy, and prove compliance across every project.",
    images: ["/og.png"],
  },
  icons: {
    // `icon` omitted on purpose: src/app/favicon.ico is auto-detected already.
    apple: "/sentinel-mark.png",
  },
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className="font-sans h-full antialiased"
    >
      <body className="min-h-full flex flex-col">
        {children}
        <Toaster />
      </body>
    </html>
  );
}
