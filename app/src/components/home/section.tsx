import { cn } from "@/lib/utils";

export function Container({ className, children }: { className?: string; children: React.ReactNode }) {
  return <div className={cn("mx-auto w-full max-w-[1240px] px-5 sm:px-8", className)}>{children}</div>;
}

/** A section heading in the wide display face with its one supporting sentence. */
export function SectionIntro({
  id,
  title,
  body,
  className,
  children,
}: {
  id: string;
  title: React.ReactNode;
  body: React.ReactNode;
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <div className={cn("flex max-w-2xl flex-col gap-5", className)}>
      <h2 id={id} className="font-display text-[clamp(2rem,4vw,3.25rem)] leading-[1.02] text-foreground">
        {title}
      </h2>
      <p className="max-w-[52ch] text-[1.0625rem] leading-relaxed text-muted-foreground">{body}</p>
      {children}
    </div>
  );
}
