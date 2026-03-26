import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold tracking-wide",
  {
    variants: {
      tone: {
        neutral: "border-border bg-muted text-foreground",
        success: "border-primary/30 bg-primary/10 text-primary",
        warn: "border-danger/30 bg-danger/10 text-danger"
      }
    },
    defaultVariants: {
      tone: "neutral"
    }
  }
);

type BadgeProps = React.HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeVariants>;

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />;
}
