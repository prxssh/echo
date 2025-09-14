import React, { useEffect, useRef } from 'react';

export type ModalProps = {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    title?: string;
    children: React.ReactNode;
};

export default function Modal({
    open,
    onOpenChange,
    title,
    children,
}: ModalProps) {
    const ref = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        if (!open) return;
        const el = ref.current;
        const focusable = el?.querySelector<HTMLElement>(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        focusable?.focus();
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onOpenChange(false);
        };
        document.addEventListener('keydown', onKey);
        return () => document.removeEventListener('keydown', onKey);
    }, [open, onOpenChange]);

    if (!open) return null;
    return (
        <div
            role="dialog"
            aria-modal="true"
            aria-label={title || 'Dialog'}
            className="modal-root"
            style={{
                position: 'fixed',
                inset: 0,
                zIndex: 1200,
                display: 'grid',
                placeItems: 'center',
            }}
            onClick={() => onOpenChange(false)}
        >
            <div
                ref={ref}
                className="ui-card"
                style={{
                    minWidth: 320,
                    maxWidth: '90vw',
                    padding: 'var(--space-4)',
                }}
                onClick={(e) => e.stopPropagation()}
            >
                {title && (
                    <h2 className="card-title" style={{ marginTop: 0 }}>
                        {title}
                    </h2>
                )}
                {children}
            </div>
        </div>
    );
}
