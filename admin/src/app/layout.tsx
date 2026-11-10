import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Toaster } from "@ai-governance/console-ui/ui/sonner";
import { ThemeProvider } from "@ai-governance/console-ui/providers/theme-provider";

export const metadata: Metadata = {
  title: "Platform console · Tollseal",
  description: "Organizations, users, earnings, service health and audit for this Tollseal deployment.",
  robots: { index: false, follow: false },
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#edf1ee" },
    { media: "(prefers-color-scheme: dark)", color: "#12102a" },
  ],
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="font-sans h-full antialiased" suppressHydrationWarning>
      <body className="min-h-full" suppressHydrationWarning>
        <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
          {children}
        </ThemeProvider>
        <Toaster position="bottom-right" />
      </body>
    </html>
  );
}
