import React, { useEffect, useMemo, useRef, useState } from 'react';

type Props = {
    urls: string[];
    stats?: Record<string, Stat>;
};

type Stat = {
    seeders: number;
    leechers: number;
    intervalSec: number; // next announce in seconds (from ns)
    minIntervalSec: number; // min announce in seconds (from ns)
    peersCount: number;
    at: number; // timestamp ms
};

export const TrackersList: React.FC<Props> = ({ urls, stats = {} }) => {
    // Column width management for tracker grid (independent of torrent table)
    const headerRef = useRef<HTMLDivElement | null>(null);
    const dragging = useRef<{
        col: number;
        startX: number;
        startW: number;
    } | null>(null);
    const [isResizing, setIsResizing] = useState(false);

    const getVarPx = (name: string, fallback: number): number => {
        const v = getComputedStyle(document.documentElement).getPropertyValue(
            name
        );
        const px = parseInt(v || '', 10);
        return Number.isFinite(px) ? px : fallback;
    };
    const setVarPx = (name: string, value: number) => {
        document.documentElement.style.setProperty(
            name,
            `${Math.max(40, Math.round(value))}px`
        );
    };
    const persistWidths = () => {
        try {
            const widths: Record<string, number> = {};
            for (let i = 1; i <= 7; i++) {
                const v = getVarPx(`--tracker-col-${i}`, 0);
                if (v > 0) widths[i] = v;
            }
            localStorage.setItem(
                'trackerGrid.colWidths',
                JSON.stringify(widths)
            );
        } catch {}
    };
    const loadPersisted = () => {
        try {
            const raw = localStorage.getItem('trackerGrid.colWidths');
            if (!raw) return false;
            const obj = JSON.parse(raw) as Record<string, number>;
            for (const k of Object.keys(obj)) {
                const i = Number(k);
                if (!Number.isFinite(i)) continue;
                setVarPx(`--tracker-col-${i}`, obj[k]!);
            }
            return true;
        } catch {
            return false;
        }
    };
    useEffect(() => {
        // Initialize widths: apply persisted widths if present; otherwise
        // fall back to torrent column vars via CSS (handled in styles).
        loadPersisted();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    useEffect(() => {
        const onMove = (e: MouseEvent) => {
            if (!dragging.current) return;
            const { col, startX, startW } = dragging.current;
            const dx = e.clientX - startX;
            const next = Math.max(50, startW + dx);
            setVarPx(`--tracker-col-${col}`, next);
            setIsResizing(true);
        };
        const onUp = () => {
            if (dragging.current) persistWidths();
            dragging.current = null;
            setIsResizing(false);
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
        const startW = getVarPx(`--tracker-col-${colIndex}`, 120);
        dragging.current = { col: colIndex, startX: ev.clientX, startW };
        setIsResizing(true);
        document.body.style.cursor = 'col-resize';
    };
    const normalize = (u: string): string => {
        try {
            const parsed = new URL(u);
            // Lowercase scheme + host, drop query/hash, trim trailing slashes in path
            const scheme = parsed.protocol.toLowerCase();
            const host = parsed.host.toLowerCase();
            let path = parsed.pathname || '';
            path = path.replace(/\/+$/, '');
            return `${scheme}//${host}${path}`;
        } catch {
            // Fallback: trim trailing slashes
            return u.replace(/\/+$/, '');
        }
    };

    // Subscribe to announce events and update stats for known URLs
    const rows = useMemo(() => {
        return urls.map((u) => ({ url: u, data: stats[normalize(u)] }));
    }, [urls, stats]);

    // Force a layout recalculation by remounting the table when stats change.
    // This helps browsers that don't recalc table column widths until a resize.
    const tableKey = useMemo(() => {
        try {
            const keys = urls.map((u) => normalize(u));
            let acc = '';
            for (const k of keys) {
                const at = stats[k]?.at || 0;
                acc += k + ':' + at + ';';
            }
            return acc;
        } catch {
            return String(Date.now());
        }
    }, [urls, stats]);

    return (
        <div className="trackers">
            <div className="tracker-table-wrap">
                <div
                    className="tracker-grid tracker-grid-header"
                    ref={headerRef}
                >
                    <div aria-hidden="true"></div>
                    <div className="grid-resizable">
                        Announce URL
                        <div
                            className="col-resizer"
                            onMouseDown={(e) => startResize(2, e)}
                        />
                    </div>
                    <div className="num grid-resizable">
                        Seeders
                        <div
                            className="col-resizer"
                            onMouseDown={(e) => startResize(3, e)}
                        />
                    </div>
                    <div className="num grid-resizable">
                        Leechers
                        <div
                            className="col-resizer"
                            onMouseDown={(e) => startResize(4, e)}
                        />
                    </div>
                    <div className="num grid-resizable">
                        Next Announce
                        <div
                            className="col-resizer"
                            onMouseDown={(e) => startResize(5, e)}
                        />
                    </div>
                    <div className="num grid-resizable">
                        Min Announce
                        <div
                            className="col-resizer"
                            onMouseDown={(e) => startResize(6, e)}
                        />
                    </div>
                    <div className="num grid-resizable">
                        Peers
                        <div
                            className="col-resizer"
                            onMouseDown={(e) => startResize(7, e)}
                        />
                    </div>
                </div>
                {rows.map(({ url, data }) => (
                    <div key={url} className="tracker-grid tracker-grid-row">
                        <div aria-hidden="true"></div>
                        <div className="wrap" title={url}>
                            {url}
                        </div>
                        <div className="num">{data ? data.seeders : '-'}</div>
                        <div className="num">{data ? data.leechers : '-'}</div>
                        <div className="num">
                            {(() => {
                                if (!data) return '-';
                                const m = Math.floor(
                                    (data.intervalSec || 0) / 60
                                );
                                return m > 0 ? `${m}m` : '-';
                            })()}
                        </div>
                        <div className="num">
                            {(() => {
                                if (!data) return '-';
                                const m = Math.floor(
                                    (data.minIntervalSec || 0) / 60
                                );
                                return m > 0 ? `${m}m` : '-';
                            })()}
                        </div>
                        <div className="num">
                            {data ? data.peersCount : '-'}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default TrackersList;
