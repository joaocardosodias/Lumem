import './Input.css'

function Input({
  id,
  label,
  type = 'text',
  value,
  onChange,
  placeholder,
  autoComplete,
  required,
  step,
  min,
  max,
  name,
  ...rest
}) {
  return (
    <label className="ui-input" htmlFor={id}>
      <span>{label}</span>
      <input
        id={id}
        name={name ?? id}
        type={type}
        value={value}
        placeholder={placeholder}
        autoComplete={autoComplete}
        required={required}
        step={step}
        min={min}
        max={max}
        onChange={onChange}
        {...rest}
      />
    </label>
  )
}

export default Input
