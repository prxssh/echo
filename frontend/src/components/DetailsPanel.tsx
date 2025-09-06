import React, { useState } from 'react';
import { torrent as Models } from '../../wailsjs/go/models';
import { infoHashHex } from '../utils/torrent';
import FileTree from './FileTree';
import { formatBytes } from '../utils/torrent';
import TrackersList from './TrackersList';

type Props = {
    torrent: Models.Torrent;
    activeTab: 'general' | 'trackers' | 'files';
    onTabChange: (tab: 'general' | 'trackers' | 'files') => void;
};

export const DetailsPanel: React.FC<Props> = ({
    torrent: t,
    activeTab,
    onTabChange,
}) => {
    const [copied, setCopied] = useState(false);
    const id = infoHashHex(t);
    const name = t.metainfo?.info?.name || 'Unnamed torrent';
    const sizeStr = formatBytes(t.metainfo?.size || 0);
    const pieces = t.metainfo?.info?.pieces?.length || 0;
    const pieceLenStr = formatBytes(t.metainfo?.info?.pieceLength || 0);
    const created = t.metainfo?.creationDate
        ? new Date(t.metainfo.creationDate as any).toLocaleString()
        : '';
    const comment = t.metainfo?.comment || '';
    const trackers = t.metainfo?.announceUrls?.length || 0;
    const files = t.metainfo?.info?.files?.length || 0;
    const isPrivate = !!t.metainfo?.info?.private;

    return (
        <div className="card" style={{ marginTop: 12 }}>
            <div className="tabs">
                <button
                    className={`tab ${activeTab === 'general' ? 'active' : ''}`}
                    onClick={() => onTabChange('general')}
                >
                    General
                </button>
                <button
                    className={`tab ${activeTab === 'trackers' ? 'active' : ''}`}
                    onClick={() => onTabChange('trackers')}
                >
                    Trackers
                </button>
                <button
                    className={`tab ${activeTab === 'files' ? 'active' : ''}`}
                    onClick={() => onTabChange('files')}
                >
                    Files
                </button>
            </div>

            {activeTab === 'general' && (
                <div className="general">
                    <div className="metrics">
                        <div className="metric">
                            <div className="metric-label">Size</div>
                            <div className="metric-value">{sizeStr}</div>
                        </div>
                        <div className="metric">
                            <div className="metric-label">Pieces</div>
                            <div className="metric-value">{pieces}</div>
                        </div>
                        <div className="metric">
                            <div className="metric-label">Piece Size</div>
                            <div className="metric-value">{pieceLenStr}</div>
                        </div>
                        <div className="metric">
                            <div className="metric-label">Privacy</div>
                            <div
                                className={`badge ${isPrivate ? 'badge-warn' : 'badge-ok'}`}
                                aria-label={
                                    isPrivate
                                        ? 'Private torrent'
                                        : 'Public torrent'
                                }
                            >
                                {isPrivate ? 'Private' : 'Public'}
                            </div>
                        </div>
                    </div>

                    <div className="kv name-row">
                        <div className="label">Name</div>
                        <div className="value wrap">{name}</div>
                    </div>

                    <div className="id-block">
                        <div className="label">Infohash</div>
                        <div className="id-row">
                            <div className="mono wrap break-all">{id}</div>
                            <button
                                className="btn-ghost btn-copy"
                                title="Copy infohash"
                                aria-label="Copy infohash"
                                onClick={() => {
                                    if (
                                        navigator.clipboard &&
                                        'writeText' in navigator.clipboard
                                    ) {
                                        navigator.clipboard
                                            .writeText(id)
                                            .then(() => {
                                                setCopied(true);
                                                setTimeout(
                                                    () => setCopied(false),
                                                    1500
                                                );
                                            });
                                    }
                                }}
                            >
                                Copy
                            </button>
                        </div>
                        {copied && (
                            <div
                                className="toast"
                                role="status"
                                aria-live="polite"
                            >
                                Copied
                            </div>
                        )}
                    </div>

                    <div className="kv-grid">
                        {created && (
                            <div className="kv">
                                <div className="label">Created</div>
                                <div className="value">{created}</div>
                            </div>
                        )}
                        <button
                            className="kv kv-link"
                            onClick={() => onTabChange('trackers')}
                            aria-label="View trackers"
                        >
                            <div className="label">Trackers</div>
                            <div className="value">{trackers}</div>
                        </button>
                        <button
                            className="kv kv-link"
                            onClick={() => onTabChange('files')}
                            aria-label="View files"
                        >
                            <div className="label">Files</div>
                            <div className="value">{files}</div>
                        </button>
                    </div>

                    {comment && (
                        <div className="section-block">
                            <div className="label">Comment</div>
                            <div className="wrap">{comment}</div>
                        </div>
                    )}
                </div>
            )}

            {activeTab === 'trackers' &&
                (t.metainfo?.announceUrls?.length || 0) > 0 && (
                    <div className="section-block">
                        <TrackersList urls={t.metainfo?.announceUrls || []} />
                    </div>
                )}

            {activeTab === 'files' &&
                (t.metainfo?.info?.files?.length || 0) > 0 && (
                    <div className="section-block">
                        <FileTree
                            files={(t.metainfo?.info?.files as any) || []}
                        />
                    </div>
                )}
        </div>
    );
};

export default DetailsPanel;
