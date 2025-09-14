import React from 'react';

export type InputProps = React.InputHTMLAttributes<HTMLInputElement> & {
    label?: string;
    error?: string;
    helperText?: string;
};

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
    ({ label, error, helperText, id, className = '', ...rest }, ref) => {
        const inputId = id || React.useId();
        const describedBy =
            [helperText ? `${inputId}-help` : '', error ? `${inputId}-err` : '']
                .filter(Boolean)
                .join(' ') || undefined;
        return (
            <div>
                {label && (
                    <label
                        htmlFor={inputId}
                        className="label"
                        style={{ display: 'block', marginBottom: '6px' }}
                    >
                        {label}
                    </label>
                )}
                <input
                    id={inputId}
                    ref={ref}
                    className={`ui-input ${className}`.trim()}
                    aria-invalid={!!error || undefined}
                    aria-describedby={describedBy}
                    {...rest}
                />
                {helperText && (
                    <div
                        id={`${inputId}-help`}
                        className="muted"
                        style={{ marginTop: 4 }}
                    >
                        {helperText}
                    </div>
                )}
                {error && (
                    <div
                        id={`${inputId}-err`}
                        role="alert"
                        style={{ marginTop: 4, color: '#ff6b6b' }}
                    >
                        {error}
                    </div>
                )}
            </div>
        );
    }
);

Input.displayName = 'Input';

export default Input;
