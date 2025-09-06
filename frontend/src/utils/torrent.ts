import { torrent as Models } from '../../wailsjs/go/models';

export function infoHashHex(t: Models.Torrent): string {
    const arr = t.metainfo?.info?.infoHash as number[] | undefined;
    if (!arr || arr.length === 0) return '';
    let s = '';
    for (let i = 0; i < arr.length; i++)
        s += (arr[i] & 0xff).toString(16).padStart(2, '0');
    return s;
}

export type TorrentRow = {
    id: string;
    name: string;
    size: string;
    sizeBytes: number;
    pieces: number;
    pieceLen: string;
    pieceLenBytes: number;
    trackers: number; // retained for details panel
    isPrivate: boolean; // retained for details panel
};

export function toRow(t: Models.Torrent): TorrentRow {
    const id = infoHashHex(t);
    const name = t.metainfo?.info?.name || 'Unnamed torrent';
    const sizeBytes = t.metainfo?.size || 0;
    const size = formatBytes(sizeBytes);
    const pieces = t.metainfo?.info?.pieces?.length || 0;
    const pieceLenBytes = t.metainfo?.info?.pieceLength || 0;
    const pieceLen = formatBytes(pieceLenBytes);
    const trackers = t.metainfo?.announceUrls?.length || 0;
    const isPrivate = !!t.metainfo?.info?.private;
    return {
        id,
        name,
        size,
        sizeBytes,
        pieces,
        pieceLen,
        pieceLenBytes,
        trackers,
        isPrivate,
    };
}

export function formatBytes(bytes: number, decimals = 2): string {
    if (!Number.isFinite(bytes) || bytes < 0) return '0 B';
    const k = 1024;
    if (bytes === 0) return '0 B';
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    const value = bytes / Math.pow(k, i);
    const fixed = i === 0 ? 0 : dm; // no decimals for bytes
    return `${value.toFixed(fixed)} ${sizes[i]}`;
}
