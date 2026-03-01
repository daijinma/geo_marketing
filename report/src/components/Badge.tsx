import { ReactNode } from 'react';

type BadgeProps = {
  children: ReactNode;
  className?: string;
};

export function Badge({ children, className = '' }: BadgeProps) {
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 ${className}`}>
      {children}
    </span>
  );
}
