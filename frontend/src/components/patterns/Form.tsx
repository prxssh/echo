import React from 'react';
import Input, { InputProps } from '../primitives/Input';

export type FormFieldProps = InputProps & {
    name: string;
    required?: boolean;
};

export function FormField(props: FormFieldProps) {
    return <Input {...props} />;
}

export default function Form({
    children,
    onSubmit,
}: React.FormHTMLAttributes<HTMLFormElement>) {
    return (
        <form onSubmit={onSubmit} noValidate>
            <div
                className="ui-stack"
                style={{ flexDirection: 'column', gap: 'var(--space-3)' }}
            >
                {children}
            </div>
        </form>
    );
}
