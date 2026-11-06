import {render, screen} from '@testing-library/react'
import {DiffView} from './DiffView'

const change = {id: 1, detectedAt: '', acknowledgedAt: '', summary: '', prevHash: 'a', newHash: 'b'}

describe('DiffView', () => {
  it('marks added and removed lines', () => {
    render(<DiffView detail={{change, fields: [], lines: [
      {op: 'equal', text: 'same'}, {op: 'removed', text: 'old'}, {op: 'added', text: 'new'},
    ]}} />)
    expect(screen.getByLabelText('removed: old')).toHaveClass('removed')
    expect(screen.getByLabelText('added: new')).toHaveClass('added')
  })
  it('renders fetched markup as text, never as HTML (NFR-SEC-002)', () => {
    const {container} = render(<DiffView detail={{change, fields: [], lines: [{op: 'added', text: '<img src=x onerror=alert(1)>'}]}} />)
    expect(container.querySelector('img')).toBeNull()
    expect(screen.getByText('<img src=x onerror=alert(1)>')).toBeInTheDocument()
  })
  it('lists changed fields', () => {
    render(<DiffView detail={{change, lines: [], fields: [{name: 'Availability', old: 'In stock', new: ''}]}} />)
    expect(screen.getByText('Availability')).toBeInTheDocument()
    expect(screen.getByText('(empty)')).toBeInTheDocument()
  })
})
