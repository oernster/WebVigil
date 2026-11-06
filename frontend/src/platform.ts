// What differs between Windows, macOS and Linux, worded in one place. Go names the platform in
// AppStateDTO.platform; only Windows has a tray, so only there does closing the window offer to keep
// WebVigil running out of sight.

export interface PlatformWords {
  hasTray: boolean
  // followSystem is the Appearance choice that tracks the operating system's own light or dark.
  followSystem: string
  signIn: string
  // closing is the Guide's account of the close button and of where monitoring runs.
  closing: string
}

const windowsWords: PlatformWords = {
  hasTray: true,
  followSystem: 'Follow Windows',
  signIn: 'Start WebVigil when I sign in to Windows',
  closing: 'Closing the window asks whether to minimise WebVigil to the tray or to quit. In the tray, monitoring carries on and the icon\'s menu offers Open, Refresh All, Pause or Resume and Exit. Quit and Exit stop monitoring.',
}

// Off Windows there is no tray: closing the window quits, so the checks run while it is open.
const noTrayClosing = 'Closing the window quits WebVigil and stops monitoring; leave it open or minimised for the checks to run.'

const words: Record<string, PlatformWords> = {
  windows: windowsWords,
  macos: {hasTray: false, followSystem: 'Follow macOS', signIn: 'Start WebVigil when I log in', closing: noTrayClosing},
  linux: {hasTray: false, followSystem: 'Follow the desktop', signIn: 'Start WebVigil when I log in', closing: noTrayClosing},
}

// platformWords answers the wording for a platform key. Before the state has arrived the key is
// empty; Windows wording stands in, since nothing can be closed or saved before then.
export function platformWords(platform: string): PlatformWords {
  return words[platform] ?? windowsWords
}
