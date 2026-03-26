import * as React from "react";
import { cn } from "@/lib/utils";

export function Panel({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <section
      className={cn(
        "rounded-2xl border border-border/70 bg-card/80 p-5 backdrop-blur-sm supports-[backdrop-filter]:bg-card/70",
        className
      )}
      {...props}
    />
  );
}
