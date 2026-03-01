'use client';

import { useId } from 'react';

interface SwitchProps {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  description?: string;
}

export function Switch({ label, checked, onChange, description }: SwitchProps) {
  const id = useId();
  const descId = description ? `${id}-desc` : undefined;

  return (
    <label className="switch" htmlFor={id}>
      <div>
        <span className="form-label" id={`${id}-label`}>{label}</span>
        {description && (
          <span
            id={descId}
            style={{ display: 'block', fontSize: '0.75rem', color: 'var(--text-tertiary)', marginTop: '0.25rem' }}
          >
            {description}
          </span>
        )}
      </div>
      <input
        id={id}
        type="checkbox"
        role="switch"
        className="switch__input"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        aria-checked={checked}
        aria-labelledby={`${id}-label`}
        aria-describedby={descId}
      />
      <span className="switch__track" aria-hidden="true" />
    </label>
  );
}
