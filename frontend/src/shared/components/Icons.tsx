/** Shared SVG icon components to avoid duplication across features */

type IconProps = {
  className?: string
  size?: number
}

/** Close / X icon — used in modals, dismiss buttons */
export function CloseIcon({ className, size = 24 }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 24 24" width={size} height={size} fill="none" stroke="currentColor" strokeWidth="2">
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  )
}

/** Chat bubble icon — used in contacts, menus */
export function ChatBubbleIcon({ className, size = 24 }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 24 24" width={size} height={size} fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    </svg>
  )
}
