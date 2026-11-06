import {render, screen} from '@testing-library/react'
import {StatusLight} from './StatusLight'

describe('StatusLight', () => {
  it('names every light in words with its own shape (FR-072)', () => {
    const shapes = new Set<string>()
    for (const [light, word] of [['grey', 'Pending'], ['green', 'Healthy'], ['amber', 'Changed'], ['red', 'Failed']]) {
      const {container, unmount} = render(<StatusLight light={light} running={false} pending={false} />)
      expect(screen.getByText(word)).toBeInTheDocument()
      shapes.add(container.querySelector('.mark')!.className)
      unmount()
    }
    expect(shapes.size).toBe(4)
  })
  it('keeps a pending change visible on a failed site (FR-065)', () => {
    render(<StatusLight light="red" running={false} pending />)
    expect(screen.getByText('Failed')).toBeInTheDocument()
    expect(screen.getByLabelText('change awaiting acknowledgement')).toBeInTheDocument()
  })
  it('shows a separate checking indicator', () => {
    render(<StatusLight light="green" running pending={false} />)
    expect(screen.getByRole('status', {name: 'checking'})).toBeInTheDocument()
  })
})
