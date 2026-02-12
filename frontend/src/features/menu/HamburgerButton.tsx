type Props = {
  onClick: () => void
  isOpen?: boolean
}

export function HamburgerButton({ onClick, isOpen = false }: Props) {
  return (
    <button
      className={`hamburger-btn ${isOpen ? 'open' : ''}`}
      onClick={onClick}
      aria-label="Toggle menu"
      aria-expanded={isOpen}
    >
      <span className="hamburger-line" />
      <span className="hamburger-line" />
      <span className="hamburger-line" />
    </button>
  )
}
