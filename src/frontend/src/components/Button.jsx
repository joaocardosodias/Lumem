import './Button.css'

function Button({ children, type = 'button', variant = 'solid', onClick }) {
  return (
    <button type={type} className={`ui-button ui-button--${variant}`} onClick={onClick}>
      {children}
    </button>
  )
}

export default Button
