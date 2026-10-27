import * as React from "react"
import { cn } from "@/lib/utils"

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "h-9 w-full min-w-0 rounded-[2px] border border-rule bg-paper px-3 py-1 text-base shadow-none transition-[color,box-shadow] outline-none selection:bg-accent selection:text-paper file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-ink placeholder:text-ink-muted disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm ",
        "focus-visible:border-accent focus-visible:outline-2  focus-visible:outline-accent ",
        "aria-invalid:border-accent aria-invalid:ring-accent/20 ",
        className
      )}
      {...props}
    />
  )
}

export { Input }
