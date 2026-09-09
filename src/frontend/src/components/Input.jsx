import './Input.css'

function Input({ id, label, type = 'text', value, onChange, placeholder, autoComplete }) {
  return (
    <label className="ui-input" htmlFor={id}>
      <span>{label}</span>
      <input
        id={id}
        type={type}
        value={value}
        placeholder={placeholder}
        autoComplete={autoComplete}
        onChange={onChange}
      />
    </label>
  )
}

export default Input
