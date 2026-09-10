import './CadastrarButton.css'

function CadastrarButton({ children, type = 'button', variant = 'primary', onClick }) {
  const variantClass = variant === 'secondary' ? ' cadastrar-button--secondary' : ''
  return (
    <button type={type} className={`cadastrar-button${variantClass}`} onClick={onClick}>
      {children}
    </button>
  )
}

export default CadastrarButton
