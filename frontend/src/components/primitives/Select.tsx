import React from 'react';

export type SelectProps = React.SelectHTMLAttributes<HTMLSelectElement> & {
    label?: string;
    error?: string;
    helperText?: string;
    options?: { value: string; label: string }[];
};

export const Select = React.forwardRef<HTMLSelectElement, SelectProps>(
    (
        {
            label,
            error,
            helperText,
            id,
            className = '',
            options,
            children,
            ...rest
        },
        ref
    ) => {
        const selectId = id || React.useId();
        const describedBy =
            [
                helperText ? `${selectId}-help` : '',
                error ? `${selectId}-err` : '',
            ]
                .filter(Boolean)
                .join(' ') || undefined;
        return (
            <div>
                {label && (
                    <label
                        htmlFor={selectId}
                        className="label"
                        style={{ display: 'block', marginBottom: '6px' }}
                    >
                        {label}
                    </label>
                )}
                <select
                    id={selectId}
                    ref={ref}
                    className={`ui-select ${className}`.trim()}
                    aria-invalid={!!error || undefined}
                    aria-describedby={describedBy}
                    {...rest}
                >
                    {options
                        ? options.map((o) => (
                              <option key={o.value} value={o.value}>
                                  {o.label}
                              </option>
                          ))
                        : children}
                </select>
                {helperText && (
                    <div
                        id={`${selectId}-help`}
                        className="muted"
                        style={{ marginTop: 4 }}
                    >
                        {helperText}
                    </div>
                )}
                {error && (
                    <div
                        id={`${selectId}-err`}
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

Select.displayName = 'Select';

export default Select;
