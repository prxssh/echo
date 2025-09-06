import React, { useCallback, useRef, useState } from 'react';

type Props = {
    onSelect?: (files: File[]) => void;
    multiple?: boolean;
    hint?: string;
};

function isTorrent(file: File): boolean {
    const lowered = file.name.toLowerCase();
    return (
        lowered.endsWith('.torrent') || file.type === 'application/x-bittorrent'
    );
}

export const TorrentUploader: React.FC<Props> = ({
    onSelect,
    multiple = true,
    hint,
}) => {
    const [files, setFiles] = useState<File[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [isDragging, setIsDragging] = useState(false);
    const inputRef = useRef<HTMLInputElement | null>(null);

    const handleFiles = useCallback(
        (selected: FileList | null) => {
            if (!selected) return;
            const arr = Array.from(selected);
            const torrents = arr.filter(isTorrent);
            if (torrents.length === 0) {
                setError('Please select one or more .torrent files.');
                return;
            }
            setError(null);
            setFiles(torrents);
            onSelect?.(torrents);
            // Clear selection so the same file can be chosen again and
            // avoid showing a stale "Selected N files" message.
            setFiles([]);
            if (inputRef.current) {
                inputRef.current.value = '';
            }
        },
        [onSelect]
    );

    const onDrop = useCallback(
        (e: React.DragEvent<HTMLDivElement>) => {
            e.preventDefault();
            setIsDragging(false);
            handleFiles(e.dataTransfer.files);
        },
        [handleFiles]
    );

    const onDragOver = useCallback((e: React.DragEvent<HTMLDivElement>) => {
        e.preventDefault();
        setIsDragging(true);
    }, []);

    const onDragLeave = useCallback(() => setIsDragging(false), []);

    const onClick = useCallback(() => {
        if (inputRef.current) inputRef.current.value = '';
        inputRef.current?.click();
    }, []);

    return (
        <div className="torrent-uploader">
            <div
                className={`dropzone ${isDragging ? 'dragging' : ''}`}
                onDrop={onDrop}
                onDragOver={onDragOver}
                onDragLeave={onDragLeave}
                onClick={onClick}
                role="button"
                aria-label="Upload torrent files"
                tabIndex={0}
                onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        onClick();
                    }
                }}
            >
                <div className="dropzone-text">
                    <strong>Upload torrents</strong>
                    <span>
                        {hint ??
                            'Drag & drop .torrent files here, or click to browse'}
                    </span>
                </div>
                <input
                    ref={inputRef}
                    type="file"
                    accept=".torrent,application/x-bittorrent"
                    multiple={multiple}
                    onChange={(e) => handleFiles(e.target.files)}
                    style={{ display: 'none' }}
                />
            </div>

            {error && <div className="uploader-error">{error}</div>}

            {files.length > 0 && (
                <div className="uploader-list">
                    <div className="uploader-list-header">
                        Selected {files.length} file
                        {files.length > 1 ? 's' : ''}
                    </div>
                    <ul>
                        {files.map((f, idx) => (
                            <li key={`${f.name}-${idx}`}>
                                <span className="file-name">{f.name}</span>
                                <span className="file-size">
                                    {(f.size / 1024).toFixed(1)} KB
                                </span>
                            </li>
                        ))}
                    </ul>
                </div>
            )}
        </div>
    );
};

export default TorrentUploader;
