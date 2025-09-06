import { useMemo } from 'react';
import { torrent as Models } from '../../wailsjs/go/models';
import { SortDir, SortKey } from '../components/TorrentTable';

export function useFilterSort(
    items: Models.Torrent[],
    query: string,
    sortKey: SortKey,
    sortDir: SortDir
) {
    const filtered = useMemo(() => {
        const q = query.trim().toLowerCase();
        if (!q) return items;
        return items.filter((t) => {
            const name = t.metainfo?.info?.name?.toLowerCase() || '';
            const ih =
                (t.metainfo?.info?.infoHash as number[] | undefined)
                    ?.map((b) => (b & 0xff).toString(16).padStart(2, '0'))
                    .join('') || '';
            return name.includes(q) || ih.includes(q);
        });
    }, [items, query]);

    const sorted = useMemo(() => {
        const data = [...filtered];
        data.sort((a, b) => {
            const ra = {
                name: a.metainfo?.info?.name || '',
                sizeBytes: a.metainfo?.size || 0,
                pieces: a.metainfo?.info?.pieces?.length || 0,
                pieceLenBytes: a.metainfo?.info?.pieceLength || 0,
            };
            const rb = {
                name: b.metainfo?.info?.name || '',
                sizeBytes: b.metainfo?.size || 0,
                pieces: b.metainfo?.info?.pieces?.length || 0,
                pieceLenBytes: b.metainfo?.info?.pieceLength || 0,
            };
            let cmp = 0;
            switch (sortKey) {
                case 'name':
                    cmp = ra.name.localeCompare(rb.name);
                    break;
                case 'size':
                    cmp = ra.sizeBytes - rb.sizeBytes;
                    break;
                case 'pieces':
                    cmp = ra.pieces - rb.pieces;
                    break;
                case 'pieceSize':
                    cmp = ra.pieceLenBytes - rb.pieceLenBytes;
                    break;
                default:
                    cmp = 0;
            }
            return sortDir === 'asc' ? cmp : -cmp;
        });
        return data;
    }, [filtered, sortKey, sortDir]);

    return { filtered, sorted };
}

export default useFilterSort;
