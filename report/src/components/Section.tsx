import { ReactNode } from 'react';

type SectionProps = {
  id: string;
  title: string;
  description?: string;
  children: ReactNode;
};

export function Section({ id, title, description, children }: SectionProps) {
  return (
    <section id={id} className="scroll-mt-24">
      <div className="mb-10 border-b border-border/50 pb-4">
        <h2 className="text-3xl font-serif font-bold tracking-tight text-foreground sm:text-4xl">{title}</h2>
        {description && (
          <p className="mt-3 text-lg text-muted-foreground font-light max-w-3xl">{description}</p>
        )}
      </div>
      <div className="mt-8">
        {children}
      </div>
    </section>
  );
}
