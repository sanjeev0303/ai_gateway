import { MarketingFooter } from "@/components/marketing/footer";
import { MarketingNav } from "@/components/marketing/nav";
import { LenisProvider } from "@/components/providers/lenis-provider";
import { isSignedIn } from "@/lib/session";

export default async function MarketingLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <LenisProvider>
      <div className="marketing flex min-h-full flex-col">
        <a
          href="#main"
          className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[60] focus:rounded-full focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:text-primary-foreground"
        >
          Skip to content
        </a>
        <MarketingNav signedIn={await isSignedIn()} />
        <main id="main" className="flex-1">{children}</main>
        <MarketingFooter />
      </div>
    </LenisProvider>
  );
}
