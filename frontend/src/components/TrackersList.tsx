import React, { useMemo } from 'react';

type Props = {
    urls: string[];
};

function protocol(u: string): string {
    try {
        return new URL(u).protocol.replace(':', '').toUpperCase();
    } catch {
        return 'UNKNOWN';
    }
}

export const TrackersList: React.FC<Props> = ({ urls }) => {
    const groups = useMemo(() => {
        const g: Record<string, string[]> = {};
        for (const u of urls) {
            try {
                const host = new URL(u).host || 'unknown';
                (g[host] ||= []).push(u);
            } catch {
                (g['unknown'] ||= []).push(u);
            }
        }
        return Object.entries(g).sort((a, b) => a[0].localeCompare(b[0]));
    }, [urls]);

    return (
        <div className="trackers">
            <div className="tracker-groups">
                {groups.map(([host, list]) => (
                    <div className="tracker-card" key={host}>
                        <div className="tracker-header">
                            <div className="label mono">{host}</div>
                        </div>
                        <ul className="plain-list">
                            {list.map((u, i) => (
                                <li
                                    key={`${host}-${i}`}
                                    className="tracker-row"
                                >
                                    <span
                                        className={`proto-pill proto-${protocol(u).toLowerCase()}`}
                                    >
                                        {protocol(u)}
                                    </span>
                                    <span className="mono wrap" title={u}>
                                        {u}
                                    </span>
                                </li>
                            ))}
                        </ul>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default TrackersList;
