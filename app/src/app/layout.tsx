import type { Metadata } from "next";
import "./globals.css";
import { Toaster } from "@/components/ui/sonner";
import { ThemeProvider } from "@/components/providers/theme-provider";

export const metadata: Metadata = {
  metadataBase: new URL("https://sentinel.acme.example"),
  title: "Sentinel — AI Gateway & Governance",
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
    apple: "/sentinel-mark.png",
  },
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className="font-sans h-full antialiased"
      suppressHydrationWarning
    >
      <body className="min-h-full flex flex-col" suppressHydrationWarning>
        <ThemeProvider
          attribute="class"
          defaultTheme="light"
          enableSystem
          disableTransitionOnChange
        >
          {children}
        </ThemeProvider>
        <Toaster />
      </body>
    </html>
  );
}
