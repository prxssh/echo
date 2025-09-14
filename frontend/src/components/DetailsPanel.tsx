import React, { useState } from 'react';
import Button from './primitives/Button';
import { torrent as Models } from '../../wailsjs/go/models';
import { infoHashHex } from '../utils/torrent';
import FileTree from './FileTree';
import { formatBytes } from '../utils/torrent';
import TrackersList from './TrackersList';
import Tabs from './primitives/Tabs';

type Props = {
    torrent: Models.Torrent;
    activeTab: 'general' | 'trackers';
    onTabChange: (tab: 'general' | 'trackers') => void;
    trackerStats?: Record<
        string,
        {
            seeders: number;
            leechers: number;
            intervalSec: number;
            peersCount: number;
            at: number;
        }
    >;
};

export const DetailsPanel: React.FC<Props> = ({
    torrent: t,
    activeTab,
    onTabChange,
    trackerStats,
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
        <div className="card ui-card" style={{ marginTop: 12 }}>
            <Tabs.Root value={activeTab} onValueChange={onTabChange}>
                <Tabs.List>
                    <Tabs.Trigger value="general">General</Tabs.Trigger>
                    <Tabs.Trigger value="trackers">Trackers</Tabs.Trigger>
                </Tabs.List>

                <Tabs.Panel value="general">
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
                                <div className="metric-value">
                                    {pieceLenStr}
                                </div>
                            </div>
                            <div className="metric">
                                <div className="metric-label">Privacy</div>
                                <div
                                    className="metric-value"
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
                                <Button
                                    variant="ghost"
                                    className="btn-copy"
                                    title={copied ? 'Copied' : 'Copy infohash'}
                                    aria-label={
                                        copied ? 'Copied' : 'Copy infohash'
                                    }
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
                                    {copied ? 'Copied' : 'Copy'}
                                </Button>
                            </div>
                            {/* Removed separate toast; the button label reflects Copied state */}
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
                                <div className="label">
                                    Trackers
                                    <span
                                        className="kv-affordance"
                                        aria-hidden="true"
                                    >
                                        ›
                                    </span>
                                </div>
                                <div className="value">{trackers}</div>
                            </button>
                            {comment && (
                                <div className="kv">
                                    <div className="label">Comment</div>
                                    <div className="value wrap">{comment}</div>
                                </div>
                            )}
                        </div>

                        {(t.metainfo?.info?.files?.length || 0) > 0 && (
                            <div className="section-block">
                                <div className="label">Files</div>
                                <FileTree
                                    files={
                                        (t.metainfo?.info?.files as any) || []
                                    }
                                />
                            </div>
                        )}
                    </div>
                </Tabs.Panel>

                <Tabs.Panel value="trackers">
                    {(t.metainfo?.announceUrls?.length || 0) > 0 && (
                        <div className="section-block">
                            <TrackersList
                                urls={t.metainfo?.announceUrls || []}
                                stats={trackerStats || {}}
                            />
                        </div>
                    )}
                </Tabs.Panel>
            </Tabs.Root>
        </div>
    );
};

export default DetailsPanel;
