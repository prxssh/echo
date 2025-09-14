import React, {
    createContext,
    useContext,
    useEffect,
    useMemo,
    useState,
} from 'react';
import { EventsOn } from '../../wailsjs/runtime';

export type Peer = {
    id?: string;
    addr: string;
    client?: string;
    at: number; // connected timestamp
    cc?: string; // ISO 3166-1 alpha-2, if known
    flag?: string; // emoji flag if provided
    country?: string; // localized country name if provided
    lastMsg?: string; // last message type
    lastMsgAt?: number;
    removing?: boolean;
};

type Ctx = {
    peers: Record<string, Peer>;
};

const PeersCtx = createContext<Ctx | null>(null);

function deriveId(payload: any): string {
    if (typeof payload === 'string') return payload;
    const id =
        String(payload?.id ?? '') ||
        String(payload?.peerId ?? '') ||
        String(payload?.addr ?? payload?.address ?? '');
    return id;
}

function deriveAddr(payload: any): string {
    if (typeof payload === 'string') return payload;
    const addr =
        String(payload?.addr ?? payload?.address ?? payload?.ip ?? '') ||
        [payload?.host, payload?.port].filter(Boolean).join(':');
    return addr;
}

function deriveClient(payload: any): string | undefined {
    return payload?.client || payload?.agent || payload?.ua || undefined;
}

function deriveCC(payload: any, addr: string): string | undefined {
    // Prefer explicit fields if backend provides them
    const cc = (
        payload?.cc ||
        payload?.iso ||
        payload?.isoCode ||
        payload?.countryCode ||
        ''
    )
        .toString()
        .trim();
    if (cc && /^[A-Za-z]{2}$/.test(cc)) return cc.toUpperCase();
    // Heuristic: if addr looks like a hostname with a ccTLD (e.g., x.y.z.de)
    if (/^[A-Za-z0-9.-]+$/.test(addr) && /[A-Za-z]/.test(addr)) {
        const parts = addr.split('.');
        const last = parts[parts.length - 1] || '';
        if (/^[A-Za-z]{2}$/.test(last)) return last.toUpperCase();
    }
    return undefined;
}
function deriveFlag(payload: any, cc?: string): string | undefined {
    const f = (payload?.flag || payload?.emoji || '').toString();
    if (f) return f;
    if (!cc) return undefined;
    const A = 0x1f1e6;
    const code = cc.toUpperCase();
    if (!/^[A-Z]{2}$/.test(code)) return undefined;
    const first = A + (code.charCodeAt(0) - 65);
    const second = A + (code.charCodeAt(1) - 65);
    return String.fromCodePoint(first) + String.fromCodePoint(second);
}

export function PeersProvider({ children }: { children: React.ReactNode }) {
    const [peers, setPeers] = useState<Record<string, Peer>>({});
    const timersRef = React.useRef<Map<string, number>>(new Map());
    const addTimersRef = React.useRef<Map<string, number>>(new Map());

    useEffect(() => {
        const offStart = EventsOn('peers:started', (payload: any) => {
            try {
                const id = deriveId(payload);
                const addr = deriveAddr(payload);
                if (!addr) return;
                const key = id || addr;
                // Cancel any pending removal if peer re-starts
                const t = timersRef.current.get(key);
                if (t) {
                    clearTimeout(t);
                    timersRef.current.delete(key);
                }
                setPeers((prev) => ({
                    ...prev,
                    [key]: {
                        id: id || undefined,
                        addr,
                        client: deriveClient(payload),
                        at: Date.now(),
                        cc: deriveCC(payload, addr),
                        flag: deriveFlag(payload, deriveCC(payload, addr)),
                        country:
                            (payload?.country || '').toString() || undefined,
                        removing: false,
                        lastMsg: prev[key]?.lastMsg,
                        lastMsgAt: prev[key]?.lastMsgAt,
                        // mark as added for a brief pulse
                        ...(prev[key]?.adding ? {} : { adding: true as any }),
                    },
                }));
                // Clear any previous add timer, then set a new one to remove the added pulse
                const oldAdd = addTimersRef.current.get(key);
                if (oldAdd) clearTimeout(oldAdd);
                const addTid = window.setTimeout(() => {
                    setPeers((prev) => ({
                        ...prev,
                        [key]: prev[key]
                            ? { ...prev[key]!, adding: false as any }
                            : prev[key],
                    }));
                    addTimersRef.current.delete(key);
                }, 900);
                addTimersRef.current.set(key, addTid);
            } catch {}
        });
        const offStop = EventsOn('peers:stopped', (payload: any) => {
            try {
                const addr = deriveAddr(payload);
                const id = deriveId(payload);
                const key = id || addr;
                if (!key) return;
                // Mark as removing with a pulse, then remove after delay
                setPeers((prev) => ({
                    ...prev,
                    [key]: prev[key]
                        ? { ...prev[key]!, removing: true }
                        : { addr, at: Date.now(), removing: true },
                }));
                const tid = window.setTimeout(() => {
                    setPeers((prev) => {
                        const next = { ...prev } as Record<string, Peer>;
                        delete next[key];
                        return next;
                    });
                    timersRef.current.delete(key);
                }, 3200);
                const old = timersRef.current.get(key);
                if (old) clearTimeout(old);
                timersRef.current.set(key, tid);
            } catch {}
        });
        const offMsg = EventsOn('peer:msg', (payload: any) => {
            try {
                const addr = deriveAddr(payload);
                const id = deriveId(payload);
                const key = id || addr;
                const typ = (payload?.type || '').toString();
                if (!key || !typ) return;
                setPeers((prev) => {
                    const peek = prev[key];
                    const base: Peer = peek || {
                        id: id || undefined,
                        addr,
                        at: Date.now(),
                    };
                    return {
                        ...prev,
                        [key]: {
                            ...base,
                            cc: base.cc || deriveCC(payload, addr),
                            flag:
                                base.flag ||
                                deriveFlag(
                                    payload,
                                    base.cc || deriveCC(payload, addr)
                                ),
                            country:
                                base.country ||
                                (payload?.country || '').toString() ||
                                undefined,
                            lastMsg: typ,
                            lastMsgAt: Date.now(),
                        },
                    };
                });
            } catch {}
        });
        return () => {
            if (typeof offStart === 'function') offStart();
            if (typeof offStop === 'function') offStop();
            if (typeof offMsg === 'function') offMsg();
            // Clear any pending timers
            timersRef.current.forEach((id) => clearTimeout(id));
            timersRef.current.clear();
            addTimersRef.current.forEach((id) => clearTimeout(id));
            addTimersRef.current.clear();
        };
    }, []);

    const value = useMemo<Ctx>(() => ({ peers }), [peers]);
    return <PeersCtx.Provider value={value}>{children}</PeersCtx.Provider>;
}

export function usePeers() {
    const ctx = useContext(PeersCtx);
    if (!ctx) throw new Error('usePeers must be used within PeersProvider');
    return ctx;
}
