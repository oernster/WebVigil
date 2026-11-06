import {icons} from '../icons'
import type {SiteDTO} from '../wire'
import {Modal} from './Modal'

interface Props {
  site: SiteDTO
  onDelete: () => void
  onDisableInstead: () => void
  onCancel: () => void
}

// ConfirmDelete names the site and states that its history goes with it (FR-008). Disabling is
// offered beside it, so nobody deletes a site when they only meant to stop checking it.
export function ConfirmDelete({site, onDelete, onDisableInstead, onCancel}: Props) {
  return (
    <Modal title="Delete website" icon={icons.deleteSite} narrow onClose={onCancel} footer={<>
      {site.enabled && <button className="btn left" onClick={onDisableInstead}>Disable instead</button>}
      <button className="btn primary" onClick={onCancel}>Cancel</button>
      <button className="btn danger" onClick={onDelete}>Delete</button>
    </>}>
      <p>Delete <strong>{site.name}</strong>?</p>
      <p>{site.url}</p>
      <p>This removes the website with its check history, snapshots and change records from this
        computer. It cannot be undone. Export the site first if you may want its definition back.</p>
    </Modal>
  )
}
