// Design tokens (TS helpers) — mirrors CSS variables
export const tokens = {
    color: {
        bg: 'var(--color-bg)',
        surface: 'var(--color-surface)',
        surfaceStrong: 'var(--color-surface-strong)',
        border: 'var(--color-border)',
        borderWeak: 'var(--color-border-weak)',
        text: 'var(--color-text)',
        textMuted: 'var(--color-text-muted)',
        accent: 'var(--color-accent)',
    },
    space: {
        0: 'var(--space-0)',
        1: 'var(--space-1)',
        2: 'var(--space-2)',
        3: 'var(--space-3)',
        4: 'var(--space-4)',
        5: 'var(--space-5)',
        6: 'var(--space-6)',
        7: 'var(--space-7)',
        8: 'var(--space-8)',
    },
    radius: {
        sm: 'var(--radius-1)',
        md: 'var(--radius-2)',
        lg: 'var(--radius-3)',
    },
    font: {
        ui: 'var(--font-ui)',
        mono: 'var(--font-mono)',
    },
    size: {
        xs: 'var(--font-size-xs)',
        sm: 'var(--font-size-sm)',
        md: 'var(--font-size-md)',
        lg: 'var(--font-size-lg)',
    },
    motion: {
        fast: 'var(--motion-fast)',
    },
    z: {
        overlay: 1000,
        popover: 1100,
        modal: 1200,
    },
} as const;
