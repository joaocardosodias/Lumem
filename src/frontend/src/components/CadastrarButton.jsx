import './CadastrarButton.css'

function CadastrarButton({ children, type = 'button', onClick }) {
  return (
    <button type={type} className="cadastrar-button" onClick={onClick}>
      {children}
    </button>
  )
}

export default CadastrarButton
