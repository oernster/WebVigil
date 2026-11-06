import {platformWords} from './platform'

describe('platformWords', () => {
  it('gives only Windows a tray and the choice on closing', () => {
    expect(platformWords('windows').hasTray).toBe(true)
    expect(platformWords('windows').closing).toMatch(/minimise WebVigil to the tray or to quit/)
    for (const p of ['macos', 'linux']) {
      expect(platformWords(p).hasTray).toBe(false)
      expect(platformWords(p).closing).toMatch(/quits WebVigil/)
      expect(platformWords(p).signIn).not.toMatch(/Windows/)
    }
  })
  it('names each system its own way', () => {
    expect(platformWords('windows').followSystem).toBe('Follow Windows')
    expect(platformWords('macos').followSystem).toBe('Follow macOS')
    expect(platformWords('linux').followSystem).toBe('Follow the desktop')
  })
  it('stands Windows in before the state arrives', () => {
    expect(platformWords('')).toEqual(platformWords('windows'))
  })
})
