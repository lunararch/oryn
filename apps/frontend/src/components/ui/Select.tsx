import type { SelectHTMLAttributes } from "react";

export function Select({
  className = "",
  children,
  ...props
}: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select className={`field cursor-pointer appearance-none pr-9 ${className}`} {...props}>
      {children}
    </select>
  );
}
