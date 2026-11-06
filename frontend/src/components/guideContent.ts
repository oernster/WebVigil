// The Guide's text, ported in shape from PigeonPost: held apart from the dialog that draws it so the
// component stays a renderer and the words stay one readable document. Two jobs, in this order. It
// NAMES the furniture, each entry carrying the REAL picture the bar or the list actually draws, so a
// control that is a picture can be identified by someone who has just met it. Then it states the
// handful of rules the window cannot say for itself.
//
// The status lights are not restated here: the dialog draws them from lightInfo, their one home.
//
// It is deliberately short. Anything a control says for itself (its tooltip, its own label) is left
// to the control; a guide nobody finishes explains nothing.
import donateIcon from '../assets/donate.png'
import {icons} from '../icons'
import {platformWords} from '../platform'

// GuideIconEntry is one named piece of furniture: its picture, what it is called and what it does.
export interface GuideIconEntry {
  icon: string
  name: string
  text: string
}

// GuideRule is one of the rules behind the behaviour: a claim in bold, then what it means.
export interface GuideRule {
  title: string
  text: string
}

// GuideSection is one block of the document. A section carries icon entries, the status lights,
// rules or plain paragraphs; the dialog draws whichever are present, in that order.
export interface GuideSection {
  heading: string
  intro?: string
  entries?: GuideIconEntry[]
  lights?: boolean
  rules?: GuideRule[]
  paragraphs?: string[]
}

// guideSections is the Guide for one platform: only the account of closing the window differs.
export function guideSections(platform: string): GuideSection[] {
  return [
    {
      heading: 'The bar along the top',
      intro: 'The controls read left to right: adding and changing a site, then import and export, then monitoring. Settings, the appearance toggle and Help are held at the far end. Hover any of them to see its name.',
      entries: [
        {icon: icons.addSite, name: 'Add website', text: 'a new page to watch: its address, how often to check it and what counts as a change. Preview extraction shows what WebVigil will compare before you save.'},
        {icon: icons.editSite, name: 'Edit', text: 'changes the website selected in the list. Disabled until a site is selected.'},
        {icon: icons.deleteSite, name: 'Delete', text: 'removes the selected website and its history once you confirm; disabling it instead keeps everything. Disabled until a site is selected.'},
        {icon: icons.importSites, name: 'Import sites', text: 'reads a file of site definitions and shows each one first; a site you already watch is skipped unless you choose otherwise.'},
        {icon: icons.exportSites, name: 'Export sites', text: 'writes your site definitions to a file, to carry them to another computer.'},
        {icon: icons.refresh, name: 'Refresh all', text: 'checks every enabled site now rather than waiting for its turn.'},
        {icon: icons.pause, name: 'Pause or resume', text: 'stops the scheduled checks until you resume. The button shows the action it will take, so play appears while monitoring is paused.'},
        {icon: icons.settings, name: 'Settings', text: 'the appearance, the alarm sound, repeating the alarm until acknowledged, bringing the window forward on a change and starting when you sign in.'},
        {icon: icons.darkMode, name: 'Light or dark', text: 'the toggle shows the appearance it will switch INTO, so the moon appears while the window is light.'},
        {icon: icons.helpAbout, name: 'Help', text: 'this guide, About, the licence and Check for Updates.'},
      ],
    },
    {
      heading: 'Each row of the list',
      intro: 'Click a column heading to sort by it; drag a heading\'s edge to widen it. Select a row to see its status, its changes and its check history below the list.',
      entries: [
        {icon: icons.refresh, name: 'Refresh', text: 'checks that one site now.'},
        {icon: icons.deleteSite, name: 'Delete', text: 'removes the site and its history once you confirm; disabling it instead keeps everything.'},
      ],
    },
    {
      heading: 'The status lights',
      intro: 'Each light is told by its shape and its word as well as its colour. An exclamation badge beside a light means a change is still waiting for acknowledgement.',
      lights: true,
    },
    {
      heading: 'The strip along the foot',
      entries: [
        {icon: donateIcon, name: 'Donate', text: 'opens a donation page in your own browser. Nothing in WebVigil is held back behind it.'},
      ],
    },
    {
      heading: 'Rules behind the behaviour',
      rules: [
        {
          title: 'Everything stays on this computer.',
          text: 'Your sites, their snapshots and their history live in your own user folder. There is no cloud service, no AI and no telemetry.',
        },
        {
          title: 'Beyond the pages you watch, one thing is fetched.',
          text: 'WebVigil asks GitHub whether a newer release has been published, shortly after it starts and once a day; Help > Check for Updates asks on demand. The request carries no identifier and nothing about your sites; a version you would rather not hear about again can be skipped.',
        },
        {
          title: 'The first successful check is the baseline.',
          text: 'Nothing is reported for it. A later check that differs records the change, makes the new content the baseline and raises the alarm; seeing the same content again raises nothing more, while a return to an earlier version counts as a change.',
        },
        {
          title: 'A change waits for you.',
          text: 'It stays amber until you acknowledge it in the details below the list. Acknowledging stops a repeating alarm and keeps the history.',
        },
        {
          title: 'A failure never hides a change.',
          text: 'A failed check turns the light red without clearing a change still waiting; the badge stays beside it. Failures make no sound: one warning appears once a site has failed as many times in a row as its threshold; the checks back off while it keeps failing.',
        },
        {
          title: 'Checks run while WebVigil is running.',
          text: platformWords(platform).closing,
        },
      ],
    },
    {
      heading: 'Keyboard',
      paragraphs: [
        'Tab and Shift+Tab move through the window. In the list, Up and Down move between sites and Enter selects one. On the Help button, Down, Enter or Space opens the menu and the arrows walk it.',
        'Escape closes the open dialog or menu.',
      ],
    },
  ]
}
