'use client'

import * as React from 'react'
import * as ProgressPrimitive from '@radix-ui/react-progress'

import { cn } from '../../lib/utils'

function Progress({
  className,
  value,
  ...props
}: React.ComponentProps<typeof ProgressPrimitive.Root>) {
  return (
    <ProgressPrimitive.Root
      data-slot="progress"
      className={cn(
        'bg-primary/5 relative h-4 w-full overflow-hidden rounded-none border border-primary/20',
        className,
      )}
      {...props}
    >
      <ProgressPrimitive.Indicator
        data-slot="progress-indicator"
        className="bg-primary h-full w-full flex-1 transition-all"
        style={{ 
          transform: `translateX(-${100 - (value || 0)}%)`,
          backgroundImage: 'linear-gradient(to right, transparent 0%, transparent 20%, currentColor 20%, currentColor 100%)',
          backgroundSize: '10px 100%'
        }}
      />
    </ProgressPrimitive.Root>
  )
}

export { Progress }
