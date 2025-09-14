import React, { useEffect, useMemo, useRef, useState } from 'react';
import { TorrentRow } from '../utils/torrent';

export type SortKey = 'name' | 'size' | 'pieces' | 'pieceSize';
export type SortDir = 'asc' | 'desc';

type Props = {
    rows: TorrentRow[];
    selectedId: string | null;
    onSelect: (id: string | null) => void;
    onRemove: (id: string) => void;
    sortKey?: SortKey;
    sortDir?: SortDir;
    onSort?: (key: SortKey) => void;
};

export const TorrentTable: React.FC<Props> = ({
    rows,
    selectedId,
    onSelect,
    onRemove,
    sortKey,
    sortDir,
    onSort,
}) => {
    const body = useMemo(() => rows, [rows]);

    const tableRef = useRef<HTMLTableElement | null>(null);
    const dragging = useRef<{
        col: number; // 1-based index matching --torrent-col-N
        startX: number;
        startW: number;
    } | null>(null);
    const [isResizing, setIsResizing] = useState(false);
    const isResizingRef = useRef(false);

    const getVarPx = (name: string, fallback: number): number => {
        const v = getComputedStyle(document.documentElement).getPropertyValue(name);
        const px = parseInt(v || '', 10);
        return Number.isFinite(px) ? px : fallback;
    };
    const setVarPx = (name: string, value: number) => {
        document.documentElement.style.setProperty(name, `${Math.max(40, Math.round(value))}px`);
    };
    const persistWidths = () => {
        try {
            const widths: Record<string, number> = {};
            for (let i = 1; i <= 6; i++) {
                const v = getVarPx(`--torrent-col-${i}`, 0);
                if (v > 0) widths[i] = v;
            }
            localStorage.setItem('torrentTable.colWidths', JSON.stringify(widths));
        } catch {}
    };
    const loadPersisted = () => {
        try {
            const raw = localStorage.getItem('torrentTable.colWidths');
            if (!raw) return false;
            const obj = JSON.parse(raw) as Record<string, number>;
            for (const k of Object.keys(obj)) {
                const i = Number(k);
                if (!Number.isFinite(i)) continue;
                setVarPx(`--torrent-col-${i}`, obj[k]!);
            }
            return true;
        } catch {
            return false;
        }
    };

    useEffect(() => {
        const updateVars = () => {
            if (isResizingRef.current) return; // don't fight user drag
            // If user saved sizes, don't overwrite during resize
            if (loadPersisted()) return;
            const el = tableRef.current;
            if (!el || !el.tHead || !el.tHead.rows.length) return;
            const cells = el.tHead.rows[0].cells;
            const widths = Array.from(cells).map((c) => c.getBoundingClientRect().width);
            const root = document.documentElement.style;
            for (let i = 0; i < widths.length; i++) {
                root.setProperty(`--torrent-col-${i + 1}`, `${Math.round(widths[i])}px`);
            }
        };

        updateVars();
        const ro = new ResizeObserver(() => updateVars());
        if (tableRef.current) ro.observe(tableRef.current);
        window.addEventListener('resize', updateVars);
        return () => {
            ro.disconnect();
            window.removeEventListener('resize', updateVars);
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    useEffect(() => {
        const onMove = (e: MouseEvent) => {
            if (!dragging.current) return;
            const { col, startX, startW } = dragging.current;
            const dx = e.clientX - startX;
            const next = Math.max(50, startW + dx);
            setVarPx(`--torrent-col-${col}`, next);
            setIsResizing(true);
        };
        const onUp = () => {
            if (dragging.current) {
                persistWidths();
            }
            dragging.current = null;
            setIsResizing(false);
            isResizingRef.current = false;
            document.body.style.cursor = '';
            window.removeEventListener('mousemove', onMove);
            window.removeEventListener('mouseup', onUp);
        };
        if (isResizing) {
            window.addEventListener('mousemove', onMove);
            window.addEventListener('mouseup', onUp);
        }
        return () => {
            window.removeEventListener('mousemove', onMove);
            window.removeEventListener('mouseup', onUp);
        };
    }, [isResizing]);

    const startResize = (colIndex: number, ev: React.MouseEvent) => {
        ev.preventDefault();
        const startW = getVarPx(`--torrent-col-${colIndex}`, 120);
        dragging.current = { col: colIndex, startX: ev.clientX, startW };
        setIsResizing(true);
        isResizingRef.current = true;
        document.body.style.cursor = 'col-resize';
    };

    return (
        <div className="table-wrap">
            <table ref={tableRef} className="table torrent-table">
                <colgroup>
                    <col style={{ width: 'var(--torrent-col-1, 36px)' }} />
                    <col style={{ width: 'var(--torrent-col-2, 280px)' }} />
                    <col style={{ width: 'var(--torrent-col-3, 120px)' }} />
                    <col style={{ width: 'var(--torrent-col-4, 120px)' }} />
                    <col style={{ width: 'var(--torrent-col-5, 120px)' }} />
                    <col style={{ width: 'var(--torrent-col-6, 90px)' }} />
                </colgroup>
                <thead>
                    <tr>
                        <th className="th-resizable"></th>
                        <th className="th-resizable"
                            role="columnheader"
                            aria-sort={
                                sortKey === 'name'
                                    ? sortDir === 'asc'
                                        ? 'ascending'
                                        : 'descending'
                                    : 'none'
                            }
                        >
                            <button
                                className="sort-btn"
                                onClick={() => onSort && onSort('name')}
                                title="Sort by name"
                                aria-label="Sort by name"
                            >
                                <span>Name</span>
                                <span
                                    className={`sort-indicator${sortKey === 'name' ? ' active' : ''}`}
                                >
                                    {sortKey === 'name'
                                        ? sortDir === 'asc'
                                            ? '▲'
                                            : '▼'
                                        : '↕'}
                                </span>
                            </button>
                            <div className="col-resizer" onMouseDown={(e) => startResize(2, e)} />
                        </th>
                        <th className="th-resizable"
                            role="columnheader"
                            aria-sort={
                                sortKey === 'size'
                                    ? sortDir === 'asc'
                                        ? 'ascending'
                                        : 'descending'
                                    : 'none'
                            }
                        >
                            <button
                                className="sort-btn"
                                onClick={() => onSort && onSort('size')}
                                title="Sort by size"
                                aria-label="Sort by size"
                            >
                                <span>Size</span>
                                <span
                                    className={`sort-indicator${sortKey === 'size' ? ' active' : ''}`}
                                >
                                    {sortKey === 'size'
                                        ? sortDir === 'asc'
                                            ? '▲'
                                            : '▼'
                                        : '↕'}
                                </span>
                            </button>
                            <div className="col-resizer" onMouseDown={(e) => startResize(3, e)} />
                        </th>
                        <th className="th-resizable"
                            role="columnheader"
                            aria-sort={
                                sortKey === 'pieces'
                                    ? sortDir === 'asc'
                                        ? 'ascending'
                                        : 'descending'
                                    : 'none'
                            }
                        >
                            <button
                                className="sort-btn"
                                onClick={() => onSort && onSort('pieces')}
                                title="Sort by pieces"
                                aria-label="Sort by pieces"
                            >
                                <span>Pieces</span>
                                <span
                                    className={`sort-indicator${sortKey === 'pieces' ? ' active' : ''}`}
                                >
                                    {sortKey === 'pieces'
                                        ? sortDir === 'asc'
                                            ? '▲'
                                            : '▼'
                                        : '↕'}
                                </span>
                            </button>
                            <div className="col-resizer" onMouseDown={(e) => startResize(4, e)} />
                        </th>
                        <th className="th-resizable"
                            role="columnheader"
                            aria-sort={
                                sortKey === 'pieceSize'
                                    ? sortDir === 'asc'
                                        ? 'ascending'
                                        : 'descending'
                                    : 'none'
                            }
                        >
                            <button
                                className="sort-btn"
                                onClick={() => onSort && onSort('pieceSize')}
                                title="Sort by piece size"
                                aria-label="Sort by piece size"
                            >
                                <span>Piece Size</span>
                                <span
                                    className={`sort-indicator${sortKey === 'pieceSize' ? ' active' : ''}`}
                                >
                                    {sortKey === 'pieceSize'
                                        ? sortDir === 'asc'
                                            ? '▲'
                                            : '▼'
                                        : '↕'}
                                </span>
                            </button>
                            <div className="col-resizer" onMouseDown={(e) => startResize(5, e)} />
                        </th>
                        <th className="th-resizable">Hash<div className="col-resizer" onMouseDown={(e) => startResize(6, e)} /></th>
                    </tr>
                </thead>
                <tbody>
                    {body.map((r, idx) => {
                        const isSel = selectedId === r.id;
                        return (
                            <tr
                                key={r.id}
                                className={isSel ? 'row-selected' : ''}
                                tabIndex={0}
                                role="row"
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' || e.key === ' ') {
                                        e.preventDefault();
                                        onSelect(isSel ? null : r.id);
                                    }
                                    // Basic up/down navigation
                                    if (e.key === 'ArrowDown') {
                                        const next = body[idx + 1];
                                        if (next) onSelect(next.id);
                                    }
                                    if (e.key === 'ArrowUp') {
                                        const prev = body[idx - 1];
                                        if (prev) onSelect(prev.id);
                                    }
                                }}
                                onClick={() => onSelect(isSel ? null : r.id)}
                            >
                                <td>
                                    <button
                                        className="btn-icon"
                                        title="Remove"
                                        aria-label="Remove torrent"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            onRemove(r.id);
                                        }}
                                    >
                                        ×
                                    </button>
                                </td>
                                <td className="wrap" title={r.name}>
                                    {r.name}
                                </td>
                                <td>{r.size}</td>
                                <td>{r.pieces}</td>
                                <td>{r.pieceLen}</td>
                                <td className="mono" title={r.id}>{r.id}</td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
};

export default TorrentTable;
