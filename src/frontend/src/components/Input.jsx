import './Input.css'

function Input({ id, label, type = 'text', value, onChange, placeholder }) {
  return (
    <label className="ui-input" htmlFor={id}>
      {label}
      <input
        id={id}
        type={type}
        value={value}
        placeholder={placeholder}
        onChange={onChange}
      />
    </label>
  )
}

export default Input
