import './Button.css'

function Button({
  children,
  type = 'button',
  variant = 'primary',
  className = '',
  onClick,
  ...props
}) {
  const isSecondary = variant === 'secondary' || variant === 'light'
  const variantClass = isSecondary
    ? 'ui-button--secondary cadastrar-button--secondary'
    : 'ui-button--primary ui-button--solid'

  return (
    <button
      type={type}
      className={`ui-button cadastrar-button ${variantClass} ${className}`.trim()}
      onClick={onClick}
      {...props}
    >
      {children}
    </button>
  )
}

export default Button
