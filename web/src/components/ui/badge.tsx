import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '../../lib/utils'

const badgeVariants = cva(
  'inline-flex items-center justify-center rounded-md border px-2 py-0.5 text-[11px] font-semibold w-fit whitespace-nowrap shrink-0 [&>svg]:size-3 gap-1 transition-all',
  {
    variants: {
      variant: {
        default:
          'border-primary bg-primary text-primary-foreground',
        secondary:
          'border-secondary bg-secondary text-secondary-foreground',
        destructive:
          'border-destructive bg-destructive text-white',
        outline:
          'border-primary/40 text-primary bg-transparent',
        terminal:
          'border-none bg-transparent text-primary before:content-["["] after:content-["]"] px-1 gap-0.5',
        glow:
          'bg-primary/20 text-primary border-primary/40 fx-glow',
        warning:
          'bg-accent/20 text-accent border-accent/40 fx-glow-amber',
        danger:
          'bg-destructive/20 text-destructive border-destructive/40 fx-glow-danger',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
)

function Badge({
  className,
  variant,
  asChild = false,
  ...props
}: React.ComponentProps<'span'> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot : 'span'

  return (
    <Comp
      data-slot="badge"
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
