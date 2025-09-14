import React, { useEffect, useMemo, useRef, useState } from 'react';
import { usePeers } from '../providers/PeersProvider';
import Pager from './Pager';

function fmtSince(ts: number): string {
    const sec = Math.max(0, Math.floor((Date.now() - ts) / 1000));
    const m = Math.floor(sec / 60);
    const h = Math.floor(m / 60);
    if (h > 0) return `${h}h`;
    if (m > 0) return `${m}m`;
    return '<1m';
}

function flagEmoji(cc?: string): string | undefined {
    if (!cc) return undefined;
    const A = 0x1f1e6;
    const code = cc.toUpperCase();
    if (!/^[A-Z]{2}$/.test(code)) return undefined;
    const first = A + (code.charCodeAt(0) - 65);
    const second = A + (code.charCodeAt(1) - 65);
    return String.fromCodePoint(first) + String.fromCodePoint(second);
}

export default function PeersList() {
    const { peers } = usePeers();
    const items = useMemo(() => Object.values(peers), [peers]);

    // Resizable columns (like trackers)
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
            for (let i = 1; i <= 3; i++) {
                const v = getVarPx(`--peers-col-${i}`, 0);
                if (v > 0) widths[i] = v;
            }
            localStorage.setItem('peersGrid.colWidths', JSON.stringify(widths));
        } catch {}
    };
    const loadPersisted = () => {
        try {
            const raw = localStorage.getItem('peersGrid.colWidths');
            if (!raw) return false;
            const obj = JSON.parse(raw) as Record<string, number>;
            for (const k of Object.keys(obj)) {
                const i = Number(k);
                if (!Number.isFinite(i)) continue;
                setVarPx(`--peers-col-${i}`, obj[k]!);
            }
            return true;
        } catch {
            return false;
        }
    };
    useEffect(() => {
        if (loadPersisted()) return;
        const h = headerRef.current;
        if (!h) return;
        const cells = Array.from(h.children) as HTMLElement[];
        if (!cells.length) return;
        const widths = cells.map((c) => c.getBoundingClientRect().width);
        for (let i = 0; i < widths.length; i++)
            setVarPx(`--peers-col-${i + 1}`, widths[i]);
    }, []);
    useEffect(() => {
        const onMove = (e: MouseEvent) => {
            if (!dragging.current) return;
            const { col, startX, startW } = dragging.current;
            const dx = e.clientX - startX;
            const next = Math.max(50, startW + dx);
            setVarPx(`--peers-col-${col}`, next);
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
        const startW = getVarPx(`--peers-col-${colIndex}`, 120);
        dragging.current = { col: colIndex, startX: ev.clientX, startW };
        setIsResizing(true);
        document.body.style.cursor = 'col-resize';
    };

    // Pagination
    const pageSize = 10;
    const [page, setPage] = useState(1);
    const totalPages = Math.max(1, Math.ceil(items.length / pageSize));
    const clampedPage = Math.min(page, totalPages);
    const start = (clampedPage - 1) * pageSize;
    const end = Math.min(start + pageSize, items.length);
    const pageItems = items.slice(start, end);
    if (items.length === 0) {
        return <div className="muted">No peers connected.</div>;
    }
    const hostFor = (addr: string) => {
        try {
            const [h] = addr.split(']')[0] ? addr.split(']') : [addr]; // naive IPv6 bracket handling
            const host = addr.includes(']') ? h + ']' : addr.split(':')[0];
            return host;
        } catch {
            return addr;
        }
    };

    return (
        <div className="tracker-table-wrap">
            <div className="peers-grid peers-grid-header" ref={headerRef}>
                <div className="grid-resizable">
                    Flag
                    <div
                        className="col-resizer"
                        onMouseDown={(e) => startResize(1, e)}
                    />
                </div>
                <div className="grid-resizable">
                    Address
                    <div
                        className="col-resizer"
                        onMouseDown={(e) => startResize(2, e)}
                    />
                </div>
                <div className="grid-resizable">
                    Message
                    <div
                        className="col-resizer"
                        onMouseDown={(e) => startResize(3, e)}
                    />
                </div>
            </div>
            {/* Tick to re-render so pulse can fade */}
            {pageItems.map((p) => {
                const flag = p.flag || flagEmoji(p.cc);
                const hot = p.lastMsgAt
                    ? Date.now() - p.lastMsgAt < 1200
                    : false;
                return (
                    <div
                        key={p.id || p.addr}
                        className={`peers-grid peers-grid-row ${p.adding ? 'peer-added' : ''} ${hot ? 'peer-hot' : ''} ${p.removing ? 'peer-removing' : ''}`.trim()}
                    >
                        <div title={p.country || p.cc || ''}>
                            {flag ? flag : '-'}
                        </div>
                        <div className="wrap" title={p.addr}>
                            {hostFor(p.addr)}
                        </div>
                        <div className="wrap">{p.lastMsg || '-'}</div>
                    </div>
                );
            })}
            <Pager
                page={clampedPage}
                totalPages={totalPages}
                rangeStart={items.length === 0 ? 0 : start + 1}
                rangeEnd={end}
                totalItems={items.length}
                onPrev={() => setPage((p) => Math.max(1, p - 1))}
                onNext={() => setPage((p) => Math.min(totalPages, p + 1))}
            />
        </div>
    );
}
