import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"
import { Slot } from "radix-ui"

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 rounded-[2px] text-sm font-medium whitespace-nowrap transition-all outline-none focus-visible:border-accent focus-visible:outline-2  focus-visible:outline-accent  disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-accent aria-invalid:ring-accent/20  [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "bg-accent text-paper hover:bg-accent-deep",
        destructive:
          "bg-accent text-paper hover:bg-accent-deep focus-visible:ring-accent/20  ",
        outline:
          "border border-rule bg-paper text-ink shadow-none hover:bg-ink hover:text-paper   ",
        secondary:
          "bg-paper-deep text-ink hover:bg-paper-deep/80",
        ghost:
          "hover:bg-paper-deep hover:text-ink ",
        link: "text-accent underline-offset-4 hover:underline",
      },
      size: {
        plain: "min-h-8",
        default: "h-9 px-4 py-2 has-[>svg]:px-3",
        xs: "h-6 gap-1 rounded-[2px] px-2 text-xs has-[>svg]:px-1.5 [&_svg:not([class*='size-'])]:size-3",
        sm: "h-8 gap-1.5 rounded-[2px] px-3 has-[>svg]:px-2.5",
        lg: "h-10 rounded-[2px] px-6 has-[>svg]:px-4",
        icon: "size-9",
        "icon-xs": "size-6 rounded-[2px] [&_svg:not([class*='size-'])]:size-3",
        "icon-sm": "size-8",
        "icon-lg": "size-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }) {
  const Comp = asChild ? Slot.Root : "button"

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button }
