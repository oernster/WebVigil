// Filling a site definition in from a ready-made reading (FR-058). The dialog holds the ignore
// rules as one rule per line of text, so the reading's rules are merged into those texts.
import type {ReadingDTO, SiteFormDTO} from './wire'

// The mode a new definition starts in; the only one a reading replaces: a definition the user
// has moved to another mode is theirs.
export const DEFAULT_MODE = 'extracted_text'

// filledForm returns the definition in the reading's mode with the reading's fields; a definition
// already moved out of the default mode is returned as it is.
export function filledForm(form: SiteFormDTO, reading: ReadingDTO): SiteFormDTO {
  if (form.mode !== DEFAULT_MODE) return form
  return {...form, mode: reading.mode, fields: reading.fields.map((f) => ({...f}))}
}

// mergeLines appends each rule the text does not already hold, keeping the user's own.
export function mergeLines(text: string, rules: string[]): string {
  const have = text.split('\n').filter((l) => l !== '')
  return [...have, ...rules.filter((r) => !have.includes(r))].join('\n')
}

// readingNote tells the user what was filled in and why, under the detection mode.
export function readingNote(name: string): string {
  return `${name}: only these fields are compared, so adverts and menus that change on every visit ` +
    'never count as a change. Edit or remove any of them.'
}
