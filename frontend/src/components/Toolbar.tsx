import React from 'react';
import Button from './primitives/Button';
import Input from './primitives/Input';

type Props = {
    title?: string;
    totalLabel?: string;
    query: string;
    onQueryChange: (v: string) => void;
    onClearAll?: () => void;
};

export const Toolbar: React.FC<Props> = ({
    title = 'Torrents',
    totalLabel,
    query,
    onQueryChange,
    onClearAll,
}) => {
    return (
        <div className="toolbar">
            <div className="toolbar-left">
                <h2
                    className="card-title align-with-name"
                    style={{ margin: 0 }}
                >
                    {title}
                </h2>
                {totalLabel && <div className="muted">{totalLabel}</div>}
            </div>
            <div className="toolbar-right">
                <Input
                    className="control"
                    placeholder="Search name or hash…"
                    value={query}
                    onChange={(e) => onQueryChange(e.target.value)}
                    aria-label="Search torrents"
                />
                {onClearAll && (
                    <Button
                        variant="ghost"
                        className="control"
                        onClick={onClearAll}
                        aria-label="Clear all torrents"
                    >
                        Clear all
                    </Button>
                )}
            </div>
        </div>
    );
};

export default Toolbar;
