import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '../../lib/utils'

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-none text-sm font-bold uppercase tracking-widest transition-all disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:ring-1 focus-visible:ring-primary",
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90 border-2 border-primary',
        destructive:
          'bg-destructive text-white hover:bg-destructive/90 border-2 border-destructive',
        outline:
          'border-2 border-primary bg-transparent text-primary hover:bg-primary/10',
        secondary:
          'border-2 border-secondary bg-transparent text-secondary hover:bg-secondary/10',
        ghost:
          'hover:bg-primary/10 hover:text-primary',
        link: 'text-primary underline-offset-4 hover:underline',
        terminal:
          'border-2 border-primary/60 bg-transparent text-primary hover:bg-primary hover:text-black fx-glow transition-all active:translate-y-[1px]',
      },
      size: {
        default: 'h-10 px-6 py-2',
        sm: 'h-8 gap-1.5 px-4',
        lg: 'h-12 px-10',
        icon: 'size-10',
        'icon-sm': 'size-8',
        'icon-lg': 'size-12',
      },
    },
    defaultVariants: {
      variant: 'terminal',
      size: 'default',
    },
  },
)

function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: React.ComponentProps<'button'> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }) {
  const Comp = asChild ? Slot : 'button'

  return (
    <Comp
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
