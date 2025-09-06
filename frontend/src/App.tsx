import React, { useCallback, useEffect, useMemo, useState } from 'react';
import './App.css';
import TorrentUploader from './components/TorrentUploader';
import Toolbar from './components/Toolbar';
import TorrentTable, { SortDir, SortKey } from './components/TorrentTable';
import { toRow, formatBytes } from './utils/torrent';
import Pager from './components/Pager';
import DetailsPanel from './components/DetailsPanel';
import { ParseTorrent } from '../wailsjs/go/ui/UI';
import { torrent as Models } from '../wailsjs/go/models';
import useResponsivePageSize from './hooks/useResponsivePageSize';
import useFilterSort from './hooks/useFilterSort';

// Using generated types from Wails models (see ../wailsjs/go/models)

function App() {
    const [items, setItems] = useState<Models.Torrent[]>([]);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const [activeTab, setActiveTab] = useState<
        'general' | 'trackers' | 'files'
    >('general');
    const [query, setQuery] = useState('');
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [page, setPage] = useState<number>(1);
    const pageSize = useResponsivePageSize(5, 10, 1200);
    const totalSize = useMemo(
        () => items.reduce((acc, t) => acc + (t?.metainfo?.size || 0), 0),
        [items]
    );

    const infoHashHex = (t: Models.Torrent): string => {
        const arr = t.metainfo?.info?.infoHash as number[] | undefined;
        if (!arr || arr.length === 0) return '';
        let s = '';
        for (let i = 0; i < arr.length; i++)
            s += (arr[i] & 0xff).toString(16).padStart(2, '0');
        return s;
    };

    const handleSelect = useCallback(
        async (files: File[]) => {
            setBusy(true);
            setError(null);
            try {
                const parsed: Models.Torrent[] = [];
                for (const f of files) {
                    const buf = new Uint8Array(await f.arrayBuffer());
                    const info = await ParseTorrent(Array.from(buf));
                    parsed.push(info as Models.Torrent);
                }

                // Enforce uniqueness by infohash against current list and within batch
                const existing = new Set(items.map((i) => infoHashHex(i)));
                const seen = new Set<string>();
                const unique = parsed.filter((p) => {
                    const id = infoHashHex(p);
                    if (!id || existing.has(id) || seen.has(id)) return false;
                    seen.add(id);
                    return true;
                });

                const skipped = parsed.length - unique.length;
                if (skipped > 0) {
                    setError(
                        `Skipped ${skipped} duplicate${skipped > 1 ? 's' : ''}.`
                    );
                }

                if (unique.length > 0) {
                    setItems((prev) => [...unique, ...prev]);
                    setPage(1);
                }
            } catch (e: any) {
                setError(e?.message ?? 'Failed to parse torrent');
            } finally {
                setBusy(false);
            }
        },
        [items]
    );

    const [sortKey, setSortKey] = useState<SortKey>('name');
    const [sortDir, setSortDir] = useState<SortDir>('asc');
    const { filtered, sorted } = useFilterSort(items, query, sortKey, sortDir);

    // Clear selection if it no longer exists in the filtered list
    useEffect(() => {
      if (!selectedId) return;
      const exists = filtered.some((t) => {
        const arr = t.metainfo?.info?.infoHash as number[] | undefined;
        const id = arr ? arr.map((b) => (b & 0xff).toString(16).padStart(2, '0')).join('') : '';
        return id == selectedId;
      });
      if (!exists) setSelectedId(null);
    }, [filtered, selectedId]);


    const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
    const clampedPage = Math.min(page, totalPages);
    const pageStart = (clampedPage - 1) * pageSize;
    const pageEnd = Math.min(pageStart + pageSize, filtered.length);
    // Sorting logic moved to useFilterSort

    const pageItems = sorted.slice(pageStart, pageEnd);

    return (
        <div className="page">
            {/* Intentionally no app title/header to keep UI minimal */}

            <main
                className={`content ${items.length === 0 ? 'content-center' : ''}`}
            >
                <div className="card">
                    <h2 className="card-title">Add Torrents</h2>
                    <p className="card-desc">
                        Select or drop .torrent files to get started.
                    </p>
                    <TorrentUploader
                        onSelect={handleSelect}
                        hint={busy ? 'Parsing…' : undefined}
                    />
                    {error && (
                        <div
                            className="uploader-error"
                            role="status"
                            aria-live="polite"
                            style={{ marginTop: 8 }}
                        >
                            {error}
                        </div>
                    )}
                </div>

                {items.length > 0 && (
                    <div className="card" style={{ marginTop: 16 }}>
                        <Toolbar
                            totalLabel={`${items.length} total • ${formatBytes(totalSize)}`}
                            query={query}
                            onQueryChange={setQuery}
                            onClearAll={() => {
                                setItems([]);
                                setSelectedId(null);
                            }}
                        />
                        <TorrentTable
                            rows={pageItems.map(toRow)}
                            selectedId={selectedId}
                            onSelect={setSelectedId}
                            onRemove={(id: string) => {
                                setItems((prev) =>
                                    prev.filter(
                                        (x) =>
                                            (
                                                x.metainfo?.info?.infoHash as
                                                    | number[]
                                                    | undefined
                                            )
                                                ?.map((b) =>
                                                    (b & 0xff)
                                                        .toString(16)
                                                        .padStart(2, '0')
                                                )
                                                .join('') !== id
                                    )
                                );
                                if (selectedId === id) setSelectedId(null);
                            }}
                            sortKey={sortKey}
                            sortDir={sortDir}
                            onSort={(key) => {
                                if (sortKey === key) {
                                    setSortDir(
                                        sortDir === 'asc' ? 'desc' : 'asc'
                                    );
                                } else {
                                    setSortKey(key);
                                    setSortDir('asc');
                                }
                            }}
                        />
                        <Pager
                            page={clampedPage}
                            totalPages={totalPages}
                            rangeStart={
                                filtered.length === 0 ? 0 : pageStart + 1
                            }
                            rangeEnd={pageEnd}
                            totalItems={filtered.length}
                            onPrev={() => setPage((p) => Math.max(1, p - 1))}
                            onNext={() =>
                                setPage((p) => Math.min(totalPages, p + 1))
                            }
                        />
                    </div>
                )}

                {selectedId &&
                    (() => {
                        const sel = filtered.find(
                            (t) =>
                                (
                                    t.metainfo?.info?.infoHash as
                                        | number[]
                                        | undefined
                                )
                                    ?.map((b) =>
                                        (b & 0xff).toString(16).padStart(2, '0')
                                    )
                                    .join('') === selectedId
                        );
                        if (!sel) return null;
                        return (
                            <DetailsPanel
                                torrent={sel}
                                activeTab={activeTab}
                                onTabChange={setActiveTab}
                            />
                        );
                    })()}

                {/* Inline details handled per row via selectedId */}
            </main>

            <footer className="footer">
                <span className="muted">v0.1.0</span>
            </footer>
        </div>
    );
}

export default App;
