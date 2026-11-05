import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Toaster } from "@/components/ui/sonner";
import { ThemeProvider } from "@/components/providers/theme-provider";

export const metadata: Metadata = {
  metadataBase: new URL("https://tollseal.acme.example"),
  title: "Tollseal — AI Gateway & Governance",
  description:
    "One AI gateway for every provider: connect OpenAI, Anthropic, Gemini and more once, issue gateway keys, call from the CLI, SDK or REST, with policy, budgets and a record of every request.",
  applicationName: "Tollseal",
  openGraph: {
    type: "website",
    siteName: "Tollseal",
    title: "Tollseal — AI governance console",
    description:
      "Every AI provider behind one gateway key, with policy, budgets and a record of every request.",
    url: "/",
    images: [{ url: "/og.png", width: 1200, height: 630, alt: "Tollseal" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "Tollseal — AI governance console",
    description:
      "Every AI provider behind one gateway key, with policy, budgets and a record of every request.",
    images: ["/og.png"],
  },
  icons: {
    apple: "/tollseal-mark.png",
  },
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#edf1ee" },
    { media: "(prefers-color-scheme: dark)", color: "#12102a" },
  ],
};

/*
 * Runs before first paint. With motion allowed, entrance animations start from
 * their hidden state instead of rendering complete and then hiding (a flicker
 * on every load). If the page script never takes over, everything is shown
 * after six seconds.
 */
const MOTION_BOOT = `(function(){try{var d=document.documentElement;if(new URLSearchParams(location.search).get("motion")==="static"||matchMedia("(prefers-reduced-motion: reduce)").matches)return;d.classList.add("motion-ok");setTimeout(function(){if(!d.classList.contains("motion-live"))d.classList.add("motion-done")},6000)}catch(e){}})();`;

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className="font-sans h-full antialiased"
      suppressHydrationWarning
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: MOTION_BOOT }} />
      </head>
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
