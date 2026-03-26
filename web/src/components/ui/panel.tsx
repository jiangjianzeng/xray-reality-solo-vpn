import * as React from "react";
import { cn } from "@/lib/utils";

export const Panel = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(
  ({ className, ...props }, ref) => {
    return (
      <section
        ref={ref}
        className={cn(
          "rounded-2xl border border-border/70 bg-card/80 p-5 backdrop-blur-sm supports-[backdrop-filter]:bg-card/70",
          className
        )}
        {...props}
      />
    );
  }
);

Panel.displayName = "Panel";
