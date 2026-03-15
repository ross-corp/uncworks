'use client'

import * as React from 'react'
import * as TabsPrimitive from '@radix-ui/react-tabs'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const tabsListVariants = cva(
  'inline-flex h-10 items-center justify-center p-1 transition-all',
  {
    variants: {
      variant: {
        default: 'bg-primary/10 border-2 border-primary/20 rounded-none w-full justify-start gap-1 p-1 font-mono uppercase tracking-widest text-[11px]',
        terminal: 'bg-black border-2 border-primary/20 rounded-none w-full justify-start gap-1 p-1 font-mono uppercase tracking-widest text-[10px]',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

const tabsTriggerVariants = cva(
  'inline-flex h-[calc(100%-1px)] items-center justify-center gap-1.5 px-4 py-1 text-sm font-bold whitespace-nowrap transition-all focus-visible:ring-1 focus-visible:ring-primary outline-none disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'rounded-none border-b-2 border-transparent text-primary/40 data-[state=active]:border-primary data-[state=active]:text-primary data-[state=active]:fx-glow data-[state=active]:bg-primary/10 hover:text-primary/70 h-full uppercase tracking-widest',
        terminal: 'rounded-none border-b-2 border-transparent text-primary/40 data-[state=active]:border-primary data-[state=active]:text-primary data-[state=active]:fx-glow hover:text-primary/70 px-4 h-full',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

function Tabs({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return (
    <TabsPrimitive.Root
      data-slot="tabs"
      className={cn('flex flex-col gap-2', className)}
      {...props}
    />
  )
}

function TabsList({
  className,
  variant,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.List> & VariantProps<typeof tabsListVariants>) {
  return (
    <TabsPrimitive.List
      data-slot="tabs-list"
      className={cn(tabsListVariants({ variant, className }))}
      {...props}
    />
  )
}

function TabsTrigger({
  className,
  variant,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Trigger> & VariantProps<typeof tabsTriggerVariants>) {
  return (
    <TabsPrimitive.Trigger
      data-slot="tabs-trigger"
      className={cn(tabsTriggerVariants({ variant, className }))}
      {...props}
    />
  )
}

function TabsContent({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="tabs-content"
      className={cn('flex-1 outline-none', className)}
      {...props}
    />
  )
}

export { Tabs, TabsList, TabsTrigger, TabsContent, tabsListVariants, tabsTriggerVariants }
