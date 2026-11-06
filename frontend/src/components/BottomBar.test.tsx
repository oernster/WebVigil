import {render, screen} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {api} from '../api'
import {BottomBar, DONATE_URL} from './BottomBar'

describe('BottomBar', () => {
  it('hands the one WebVigil payment address to the browser', async () => {
    const open = vi.spyOn(api, 'openExternal').mockResolvedValue(true)
    const onError = vi.fn()
    render(<BottomBar onError={onError} />)
    const button = screen.getByRole('button', {name: 'Donate to support WebVigil (opens your browser)'})
    await userEvent.click(button)
    // Asserted literally: a typo in the payment address must fail here, not send a supporter elsewhere.
    expect(open).toHaveBeenCalledWith('https://www.paypal.com/ncp/payment/4JDW9BNNG97CE', onError)
    expect(DONATE_URL.startsWith('https://')).toBe(true)
    open.mockRestore()
  })
  it('sits in a bar of its own at the foot', () => {
    const {container} = render(<BottomBar onError={vi.fn()} />)
    expect(container.querySelector('footer.toolbar.bottombar button.donate img')).not.toBeNull()
  })
})
