import { cn } from "@/lib/utils";

export function Container({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("mx-auto w-full max-w-7xl px-6 sm:px-8 lg:px-12", className)}>
      {children}
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
    <section
      id={id}
      className={cn(
        "border-b border-[rgba(111,78,55,0.12)] py-24 sm:py-32 lg:py-40",
        className
      )}
    >
      <Container className={containerClassName}>{children}</Container>
    </section>
  );
}

export function SectionLabel({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <p
      className={cn(
        "font-mono text-xs tracking-[0.14em] text-[#6F4E37] uppercase mb-4",
        className
      )}
      style={{ fontFamily: "var(--font-fira-code)" }}
    >
      {children}
    </p>
  );
}

export function DisplayHeading({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <h2
      className={cn(
        "font-display text-5xl sm:text-6xl lg:text-7xl font-semibold leading-[0.95] tracking-[-0.04em] text-[#1C1412]",
        className
      )}
      style={{ fontFamily: "var(--font-cormorant)" }}
    >
      {children}
    </h2>
  );
}

export function SectionBody({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <p
      className={cn(
        "text-base sm:text-lg leading-relaxed text-[#4A3A30] max-w-[52ch]",
        className
      )}
    >
      {children}
    </p>
  );
}

/** Legacy compat: SectionHeading as used by existing components */
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
      {eyebrow ? <SectionLabel>{eyebrow}</SectionLabel> : null}
      <h2
        className={cn(
          "font-display text-4xl sm:text-5xl font-semibold leading-none tracking-[-0.04em] text-[#1C1412]",
          titleClassName
        )}
        style={{ fontFamily: "var(--font-cormorant)" }}
      >
        {title}
      </h2>
      {body ? (
        <p className="mt-5 text-base sm:text-lg leading-relaxed text-[#4A3A30] max-w-[52ch]">
          {body}
        </p>
      ) : null}
    </div>
  );
}

export function Note({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <p
      className={cn("text-xs text-[#7A6458]", className)}
      style={{ fontFamily: "var(--font-fira-code)" }}
    >
      {children}
    </p>
  );
}

export function GlassSection({
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
    <section
      id={id}
      className={cn(
        "relative overflow-hidden border-b border-[rgba(111,78,55,0.12)] py-24 sm:py-32 lg:py-40",
        className
      )}
    >
      <Container className={cn("relative z-10", containerClassName)}>
        {children}
      </Container>
    </section>
  );
}

/** Eyebrow kept for backward compat */
export function Eyebrow({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <p
      className={cn(
        "font-mono text-xs tracking-[0.14em] text-[#6F4E37] uppercase mb-4",
        className
      )}
      style={{ fontFamily: "var(--font-fira-code)" }}
    >
      {children}
    </p>
  );
}
