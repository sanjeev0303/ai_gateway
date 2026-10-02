import { cn } from "@/lib/utils";

export function Container({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("mx-auto w-full max-w-7xl px-4 sm:px-6", className)}>
      {children}
    </div>
  );
}

export function Eyebrow({ className, children }: { className?: string; children: React.ReactNode }) {
  return (
    <p className={cn("text-xs font-medium uppercase tracking-[0.14em] text-primary", className)}>
      {children}
    </p>
  );
}

export function SectionHeading({
  eyebrow,
  title,
  body,
  className,
  titleClassName,
}: {
  eyebrow?: string;
  title: string;
  body?: string;
  className?: string;
  titleClassName?: string;
}) {
  return (
    <div className={cn("max-w-2xl", className)}>
      {eyebrow ? <Eyebrow>{eyebrow}</Eyebrow> : null}
      <h2
        className={cn(
          "mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl",
          titleClassName,
        )}
      >
        {title}
      </h2>
      {body ? (
        <p className="mt-4 text-base leading-relaxed text-muted-foreground text-pretty sm:text-lg">
          {body}
        </p>
      ) : null}
    </div>
  );
}

export function Section({
  id,
  className,
  containerClassName,
  children,
}: {
  id?: string;
  className?: string;
  containerClassName?: string;
  children: React.ReactNode;
}) {
  return (
    <section id={id} className={cn("border-b", className)}>
      <Container className={cn("py-20 sm:py-24 lg:py-28", containerClassName)}>
        {children}
      </Container>
    </section>
  );
}

export function Note({ className, children }: { className?: string; children: React.ReactNode }) {
  return <p className={cn("text-xs text-muted-foreground", className)}>{children}</p>;
}
