import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const inputVariants = cva(
  'file:text-foreground placeholder:text-muted-foreground/50 selection:bg-primary selection:text-primary-foreground h-9 w-full min-w-0 rounded-none border bg-transparent px-3 py-1 text-sm transition-all outline-none disabled:opacity-50 tracking-[0.15em] font-mono text-foreground caret-primary border-border focus-visible:border-primary focus-visible:ring-1 focus-visible:ring-primary/30',
  {
    variants: {
      variant: {
        default: '',
        terminal: 'fx-scanlines',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

function Input({
  className,
  type,
  variant,
  ...props
}: React.ComponentProps<'input'> & VariantProps<typeof inputVariants>) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(inputVariants({ variant, className }))}
      {...props}
    />
  )
}

export { Input, inputVariants }
