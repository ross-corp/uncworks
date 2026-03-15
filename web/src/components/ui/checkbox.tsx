'use client'

import * as React from 'react'
import * as CheckboxPrimitive from '@radix-ui/react-checkbox'

import { cn } from '../../lib/utils'

function Checkbox({
  className,
  ...props
}: React.ComponentProps<typeof CheckboxPrimitive.Root>) {
  return (
    <CheckboxPrimitive.Root
      data-slot="checkbox"
      className={cn(
        'peer border-primary/60 bg-black data-[state=checked]:bg-primary data-[state=checked]:text-black data-[state=checked]:border-primary focus-visible:ring-1 focus-visible:ring-primary size-5 shrink-0 rounded-none border-2 transition-none outline-none disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
      {...props}
    >
      <CheckboxPrimitive.Indicator
        data-slot="checkbox-indicator"
        className="flex items-center justify-center text-current transition-none"
      >
        <span className="font-mono text-[14px] leading-none font-bold">X</span>
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  )
}

export { Checkbox }
