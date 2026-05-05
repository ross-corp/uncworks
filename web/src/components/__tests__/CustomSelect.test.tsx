// web/src/components/__tests__/CustomSelect.test.tsx
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { render } from '@testing-library/react'
import CustomSelect from '../CustomSelect'

const OPTIONS = [
  { value: 'alpha', label: 'Alpha' },
  { value: 'beta', label: 'Beta' },
  { value: 'gamma', label: 'Gamma' },
]

describe('CustomSelect', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('calls onChange with value when an option is clicked', async () => {
    const onChange = vi.fn()
    render(
      <CustomSelect
        value=""
        onChange={onChange}
        options={OPTIONS}
      />
    )

    // Open the dropdown
    const summary = document.querySelector('summary') as HTMLElement
    fireEvent.click(summary)

    // Click the Beta option
    const betaOption = screen.getByText('Beta')
    fireEvent.click(betaOption)

    expect(onChange).toHaveBeenCalledWith('beta')
  })

  it('closes on outside click', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()

    render(
      <div>
        <CustomSelect value="" onChange={onChange} options={OPTIONS} />
        <button data-testid="outside">Outside</button>
      </div>
    )

    const summary = document.querySelector('summary') as HTMLElement
    const details = document.querySelector('details') as HTMLDetailsElement

    fireEvent.click(summary)
    // details should be open
    details.open = true

    // Click outside
    await user.click(screen.getByTestId('outside'))

    await waitFor(() => {
      expect(details.open).toBe(false)
    })
  })
})
