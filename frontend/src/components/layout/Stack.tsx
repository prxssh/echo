import React from 'react';

export type StackProps = React.HTMLAttributes<HTMLDivElement> & {
    direction?: 'row' | 'column';
    gap?: number | string; // CSS gap value or token var
    align?: React.CSSProperties['alignItems'];
    justify?: React.CSSProperties['justifyContent'];
};

export default function Stack({
    direction = 'row',
    gap = 'var(--space-2)',
    align,
    justify,
    style,
    ...rest
}: StackProps) {
    return (
        <div
            style={{
                display: 'flex',
                flexDirection: direction,
                gap,
                alignItems: align,
                justifyContent: justify,
                ...style,
            }}
            {...rest}
        />
    );
}
