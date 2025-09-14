import React, { createContext, useContext, useId } from 'react';

type TabsContextValue = {
    value: string;
    onChange: (v: string) => void;
    baseId: string;
};

const TabsCtx = createContext<TabsContextValue | null>(null);

export type TabsRootProps = {
    value: string;
    onValueChange: (v: string) => void;
    children: React.ReactNode;
};

export function TabsRoot({ value, onValueChange, children }: TabsRootProps) {
    const baseId = useId();
    return (
        <TabsCtx.Provider value={{ value, onChange: onValueChange, baseId }}>
            {children}
        </TabsCtx.Provider>
    );
}

export type TabsListProps = React.HTMLAttributes<HTMLDivElement> & {};

export function TabsList({ className = '', ...rest }: TabsListProps) {
    return (
        <div role="tablist" className={`tabs ${className}`.trim()} {...rest} />
    );
}

export type TabsTriggerProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
    value: string;
};

export function TabsTrigger({
    value,
    className = '',
    onKeyDown,
    ...rest
}: TabsTriggerProps) {
    const ctx = useContext(TabsCtx);
    if (!ctx) throw new Error('TabsTrigger must be used within TabsRoot');
    const selected = ctx.value === value;
    const id = `${ctx.baseId}-tab-${value}`;
    const panelId = `${ctx.baseId}-panel-${value}`;
    return (
        <button
            id={id}
            role="tab"
            aria-selected={selected}
            aria-controls={panelId}
            className={`tab ${selected ? 'active' : ''} ${className}`.trim()}
            tabIndex={selected ? 0 : -1}
            onClick={() => ctx.onChange(value)}
            onKeyDown={(e) => {
                // Minimal keyboard support: Left/Right navigates triggers among siblings
                if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight')
                    return onKeyDown?.(e);
                const triggers =
                    (e.currentTarget.parentElement?.querySelectorAll(
                        '[role="tab"]'
                    ) ?? []) as unknown as HTMLButtonElement[];
                const idx = Array.from(triggers).indexOf(
                    e.currentTarget as HTMLButtonElement
                );
                if (idx === -1) return;
                const nextIdx =
                    e.key === 'ArrowRight'
                        ? (idx + 1) % triggers.length
                        : (idx - 1 + triggers.length) % triggers.length;
                (triggers[nextIdx] as HTMLButtonElement).focus();
            }}
            {...rest}
        />
    );
}

export type TabsPanelProps = React.HTMLAttributes<HTMLDivElement> & {
    value: string;
};

export function TabsPanel({ value, className = '', ...rest }: TabsPanelProps) {
    const ctx = useContext(TabsCtx);
    if (!ctx) throw new Error('TabsPanel must be used within TabsRoot');
    const selected = ctx.value === value;
    const id = `${ctx.baseId}-panel-${value}`;
    const tabId = `${ctx.baseId}-tab-${value}`;
    return (
        <div
            id={id}
            role="tabpanel"
            aria-labelledby={tabId}
            hidden={!selected}
            className={className}
            {...rest}
        />
    );
}

export default {
    Root: TabsRoot,
    List: TabsList,
    Trigger: TabsTrigger,
    Panel: TabsPanel,
};
