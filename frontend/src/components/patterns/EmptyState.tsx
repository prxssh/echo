import React from 'react';
import Card from '../primitives/Card';
import Button from '../primitives/Button';

type Props = {
    title: string;
    description?: string;
    action?: { label: string; onClick: () => void };
};

export default function EmptyState({ title, description, action }: Props) {
    return (
        <Card>
            <h3 className="card-title" style={{ marginTop: 0 }}>
                {title}
            </h3>
            {description && (
                <p className="card-desc" style={{ marginTop: 4 }}>
                    {description}
                </p>
            )}
            {action && (
                <Button
                    variant="secondary"
                    onClick={action.onClick}
                    style={{ marginTop: 'var(--space-2)' }}
                >
                    {action.label}
                </Button>
            )}
        </Card>
    );
}
