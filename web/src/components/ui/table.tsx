'use client'

import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const tableVariants = cva(
  'w-full caption-bottom text-sm',
  {
    variants: {
      variant: {
        default: '',
        terminal: 'font-mono text-primary/80 border-collapse',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

function Table({
  className,
  variant,
  ...props
}: React.ComponentProps<'table'> & VariantProps<typeof tableVariants>) {
  return (
    <div
      data-slot="table-container"
      className={cn("relative w-full overflow-x-auto", variant === 'terminal' && 'fx-panel fx-scanlines bg-card/40 p-1')}
    >
      <table
        data-slot="table"
        className={cn(tableVariants({ variant, className }))}
        {...props}
      />
    </div>
  )
}

function TableHeader({ className, ...props }: React.ComponentProps<'thead'>) {
  return (
    <thead
      data-slot="table-header"
      className={cn('bg-primary/10 border-b-2 border-primary', className)}
      {...props}
    />
  )
}

function TableBody({ className, ...props }: React.ComponentProps<'tbody'>) {
  return (
    <tbody
      data-slot="table-body"
      className={cn('[&_tr]:border-b [&_tr]:border-primary/20', className)}
      {...props}
    />
  )
}

function TableFooter({ className, ...props }: React.ComponentProps<'tfoot'>) {
  return (
    <tfoot
      data-slot="table-footer"
      className={cn(
        'bg-primary/5 border-t-2 border-primary font-bold',
        className,
      )}
      {...props}
    />
  )
}

function TableRow({ className, ...props }: React.ComponentProps<'tr'>) {
  return (
    <tr
      data-slot="table-row"
      className={cn(
        'hover:bg-primary/5 transition-none',
        className,
      )}
      {...props}
    />
  )
}

function TableHead({ className, ...props }: React.ComponentProps<'th'>) {
  return (
    <th
      data-slot="table-head"
      className={cn(
        'text-primary h-12 px-4 text-left align-middle font-bold whitespace-nowrap uppercase tracking-[0.2em] text-[10px]',
        className,
      )}
      {...props}
    />
  )
}

function TableCell({ className, ...props }: React.ComponentProps<'td'>) {
  return (
    <td
      data-slot="table-cell"
      className={cn(
        'p-4 align-middle whitespace-nowrap text-secondary text-[11px] tracking-widest uppercase',
        className,
      )}
      {...props}
    />
  )
}

function TableCaption({
  className,
  ...props
}: React.ComponentProps<'caption'>) {
  return (
    <caption
      data-slot="table-caption"
      className={cn('text-muted-foreground mt-4 text-sm font-mono uppercase tracking-widest text-[9px]', className)}
      {...props}
    />
  )
}

export {
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
}
