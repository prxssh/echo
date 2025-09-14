import React, { useMemo } from 'react';

type Props = {
    urls: string[];
    stats?: Record<string, Stat>;
};

type Stat = {
    seeders: number;
    leechers: number;
    intervalSec: number; // converted from ns
    peersCount: number;
    at: number; // timestamp ms
};

export const TrackersList: React.FC<Props> = ({ urls, stats = {} }) => {
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
                <div className="tracker-grid tracker-grid-header">
                    <div aria-hidden="true"></div>
                    <div>Announce URL</div>
                    <div className="num">Seeders</div>
                    <div className="num">Leechers</div>
                    <div className="num">Interval</div>
                    <div className="num">Peers</div>
                </div>
                {rows.map(({ url, data }) => (
                    <div key={url} className="tracker-grid tracker-grid-row">
                        <div aria-hidden="true"></div>
                        <div className="mono wrap" title={url}>{url}</div>
                        <div className="num">{data ? data.seeders : '-'}</div>
                        <div className="num">{data ? data.leechers : '-'}</div>
                        <div className="num">{data ? `${data.intervalSec}s` : '-'}</div>
                        <div className="num">{data ? data.peersCount : '-'}</div>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default TrackersList;
