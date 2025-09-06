import React, { useMemo } from 'react';
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

    return (
        <div className="table-wrap">
            <table className="table">
                <thead>
                    <tr>
                        <th style={{ width: '36px' }}></th>
                        <th
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
                        </th>
                        <th
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
                        </th>
                        <th
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
                        </th>
                        <th
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
                        </th>
                        <th>Hash</th>
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
                                <td className="mono">{r.id.slice(0, 10)}…</td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
};

export default TorrentTable;
