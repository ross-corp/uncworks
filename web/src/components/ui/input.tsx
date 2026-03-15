import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const inputVariants = cva(
  'file:text-foreground placeholder:text-primary/30 selection:bg-primary selection:text-primary-foreground border-primary/40 h-10 w-full min-w-0 rounded-none border-2 bg-black px-3 py-1 text-sm transition-all outline-none disabled:opacity-50 uppercase tracking-[0.15em] font-mono text-primary caret-primary focus-visible:border-primary focus-visible:fx-glow focus-visible:ring-0',
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
