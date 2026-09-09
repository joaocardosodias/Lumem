import './Button.css'

function Button({ children, type = 'button', onClick }) {
  return (
    <button type={type} className="ui-button" onClick={onClick}>
      {children}
    </button>
  )
}

export default Button
