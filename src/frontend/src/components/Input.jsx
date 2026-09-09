import './Input.css'

function Input({ id, label, type = 'text', value, onChange, placeholder, autoComplete, required, step, min, max }) {
  return (
    <label className="ui-input" htmlFor={id}>
      <span>{label}</span>
      <input
        id={id}
        type={type}
        value={value}
        placeholder={placeholder}
        autoComplete={autoComplete}
        required={required}
        step={step}
        min={min}
        max={max}
        onChange={onChange}
      />
    </label>
  )
}

export default Input
