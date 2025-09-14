import React from 'react';

type Variant = 'primary' | 'secondary' | 'ghost' | 'destructive';
type Size = 'sm' | 'md' | 'lg';

export type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: Variant;
    size?: Size;
    leftIcon?: React.ReactNode;
    rightIcon?: React.ReactNode;
    loading?: boolean;
};

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
    (
        {
            variant = 'secondary',
            size = 'md',
            leftIcon,
            rightIcon,
            loading,
            children,
            className = '',
            disabled,
            ...rest
        },
        ref
    ) => {
        const vClass =
            variant === 'primary'
                ? 'primary'
                : variant === 'ghost'
                  ? 'ghost'
                  : variant === 'destructive'
                    ? 'destructive'
                    : '';
        const sClass = size === 'sm' ? 'btn-sm' : size === 'lg' ? 'btn-lg' : '';
        const stateDisabled = disabled || loading;
        return (
            <button
                ref={ref}
                className={`ui-btn ${vClass} ${sClass} ${className}`.trim()}
                aria-busy={loading || undefined}
                disabled={stateDisabled}
                {...rest}
            >
                {leftIcon && (
                    <span aria-hidden className="btn-icon-l">
                        {leftIcon}
                    </span>
                )}
                <span>{loading ? 'Loading…' : children}</span>
                {rightIcon && (
                    <span aria-hidden className="btn-icon-r">
                        {rightIcon}
                    </span>
                )}
            </button>
        );
    }
);

Button.displayName = 'Button';

export default Button;
