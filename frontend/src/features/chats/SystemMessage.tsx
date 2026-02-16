type Props = {
  text: string
}

export function SystemMessage({ text }: Props) {
  return (
    <div className="system-message">
      <span className="system-message__text">{text}</span>
    </div>
  )
}
