import React from 'react';

export type CardProps = React.HTMLAttributes<HTMLDivElement> & {
    as?: keyof JSX.IntrinsicElements;
    title?: string;
    actions?: React.ReactNode;
};

export default function Card({
    as: Comp = 'div',
    title,
    actions,
    className = '',
    children,
    ...rest
}: CardProps) {
    return (
        <Comp className={`ui-card card ${className}`.trim()} {...rest}>
            {(title || actions) && (
                <div
                    className="card-header"
                    style={{ marginBottom: 'var(--space-2)' }}
                >
                    {title && (
                        <h2 className="card-title" style={{ margin: 0 }}>
                            {title}
                        </h2>
                    )}
                    {actions}
                </div>
            )}
            {children}
        </Comp>
    );
}
