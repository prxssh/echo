import React, {
    createContext,
    useContext,
    useEffect,
    useMemo,
    useState,
} from 'react';
import { EventsOn } from '../../wailsjs/runtime';

export type TrackerStat = {
    seeders: number;
    leechers: number;
    intervalSec: number; // next announce in seconds
    minIntervalSec: number; // min announce in seconds
    peersCount: number;
    at: number; // timestamp ms
};

type Ctx = {
    stats: Record<string, TrackerStat>;
    normalize: (u: string) => string;
};

const TrackerStatsCtx = createContext<Ctx | null>(null);

function normalize(u: string): string {
    try {
        const parsed = new URL(u);
        const scheme = parsed.protocol.toLowerCase();
        const host = parsed.host.toLowerCase();
        let path = parsed.pathname || '';
        path = path.replace(/\/+$/, '');
        return `${scheme}//${host}${path}`;
    } catch {
        return (u || '').replace(/\/+$/, '');
    }
}

export function TrackerStatsProvider({
    children,
}: {
    children: React.ReactNode;
}) {
    const [stats, setStats] = useState<Record<string, TrackerStat>>({});

    useEffect(() => {
        const off = EventsOn('tracker:announce', (payload: any) => {
            try {
                const url = String(payload?.tracker ?? '');
                if (!url) return;
                const key = normalize(url);
                const intervalNs = Number(payload?.interval ?? 0);
                const minIntervalNs = Number(payload?.minInterval ?? 0);
                setStats((prev) => ({
                    ...prev,
                    [key]: {
                        seeders: Number(payload?.seeders ?? 0),
                        leechers: Number(payload?.leechers ?? 0),
                        peersCount: Number(payload?.peersCount ?? 0),
                        intervalSec: Math.max(0, Math.round(intervalNs / 1e9)),
                        minIntervalSec: Math.max(
                            0,
                            Math.round(minIntervalNs / 1e9)
                        ),
                        at: Date.now(),
                    },
                }));
            } catch {}
        });
        return () => {
            if (typeof off === 'function') off();
        };
    }, []);

    const value = useMemo<Ctx>(() => ({ stats, normalize }), [stats]);
    return (
        <TrackerStatsCtx.Provider value={value}>
            {children}
        </TrackerStatsCtx.Provider>
    );
}

export function useTrackerStats() {
    const ctx = useContext(TrackerStatsCtx);
    if (!ctx)
        throw new Error(
            'useTrackerStats must be used within TrackerStatsProvider'
        );
    return ctx;
}
