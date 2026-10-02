import Link from "next/link";
import { ArrowRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { APP_NAME } from "@/lib/constants";

interface CallToActionProps {
  heading?: string;
  body?: string;
  primaryLabel?: string;
  primaryHref?: string;
  secondaryLabel?: string;
  secondaryHref?: string;
  className?: string;
}

export function CallToAction({
  heading = `Put ${APP_NAME} in front of your AI traffic`,
  body = "Point one environment variable at the gateway and see your first attributed request in under a minute.",
  primaryLabel = "Open the dashboard",
  primaryHref = "/app/overview",
  secondaryLabel = "Read about the product",
  secondaryHref = "/product",
  className,
}: CallToActionProps) {
  return (
    <section className={className}>
      <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6">
        <div className="rounded-2xl border bg-card px-6 py-14 text-center sm:px-14">
          <h2 className="mx-auto max-w-2xl text-3xl font-semibold tracking-tight text-balance">
            {heading}
          </h2>
          <p className="mx-auto mt-4 max-w-xl text-lg text-muted-foreground text-pretty">
            {body}
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            <Button asChild size="lg">
              <Link href={primaryHref}>
                {primaryLabel}
                <ArrowRight data-icon="inline-end" aria-hidden />
              </Link>
            </Button>
            <Button asChild size="lg" variant="outline">
              <Link href={secondaryHref}>{secondaryLabel}</Link>
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}
