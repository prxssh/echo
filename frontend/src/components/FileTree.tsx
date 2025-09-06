import React from 'react';

export type FileEntry = { length: number; path: string[] };

type Node = {
    name: string;
    children?: Record<string, Node>;
    length?: number; // file size
    total?: number; // accumulated size for directories
    path?: string[];
};

function buildTree(files: FileEntry[]): Node {
    const root: Node = { name: '', children: {} };
    for (const f of files) {
        let cur = root;
        for (let i = 0; i < f.path.length; i++) {
            const part = f.path[i];
            cur.children = cur.children || {};
            cur.children[part] = cur.children[part] || {
                name: part,
                children: {},
            };
            cur = cur.children[part]!;
        }
        // Leaf node represents the file
        cur.length = f.length;
        cur.path = f.path;
        delete cur.children; // files have no children
    }
    // accumulate directory sizes
    const walk = (n: Node): number => {
        if (!n.children) return n.length || 0;
        let sum = 0;
        for (const child of Object.values(n.children)) sum += walk(child);
        n.total = sum;
        return sum;
    };
    walk(root);
    return root;
}

function formatSize(bytes: number): string {
    if (bytes >= 1024 * 1024 * 1024)
        return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
    if (bytes >= 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
    if (bytes >= 1024) return (bytes / 1024).toFixed(0) + ' KB';
    return bytes + ' B';
}

type Props = { files: FileEntry[] };

export const FileTree: React.FC<Props> = ({ files }) => {
    const tree = React.useMemo(() => buildTree(files), [files]);
    const [expanded, setExpanded] = React.useState<Set<string>>(
        () => new Set<string>()
    );

    const toggle = (key: string) => {
        setExpanded((prev) => {
            const n = new Set(prev);
            if (n.has(key)) n.delete(key);
            else n.add(key);
            return n;
        });
    };

    const renderNode = (
        node: Node,
        depth = 0,
        basePath: string[] = []
    ): React.ReactNode => {
        if (!node.children) {
            // file
            return (
                <li
                    className="filetree-file"
                    key={node.path!.join('/')}
                    style={{ paddingLeft: depth * 14 }}
                >
                    <span className="wrap">{node.name}</span>
                    <span className="muted">
                        {formatSize(node.length || 0)}
                    </span>
                </li>
            );
        }

        // directory
        const entries = Object.values(node.children).sort((a, b) => {
            const aDir = !!a.children,
                bDir = !!b.children;
            if (aDir !== bDir) return aDir ? -1 : 1;
            return a.name.localeCompare(b.name);
        });

        return entries.map((child) => {
            if (!child.children) {
                return renderNode(child, depth, basePath.concat(child.name));
            }
            const key = basePath.concat(child.name).join('/');
            const isOpen = expanded.has(key);
            return (
                <li className="filetree-dir" key={`dir-${key}`}>
                    <div
                        className="filetree-dirname"
                        style={{ paddingLeft: depth * 14 }}
                    >
                        <button
                            className="tree-toggle"
                            onClick={() => toggle(key)}
                            aria-label={
                                isOpen ? 'Collapse folder' : 'Expand folder'
                            }
                        >
                            {isOpen ? '▾' : '▸'}
                        </button>
                        <span className="wrap">{child.name}</span>
                        <span className="muted" style={{ marginLeft: 8 }}>
                            {formatSize(child.total || 0)}
                        </span>
                    </div>
                    {isOpen && (
                        <ul className="filetree-list">
                            {renderNode(
                                child,
                                depth + 1,
                                basePath.concat(child.name)
                            )}
                        </ul>
                    )}
                </li>
            );
        });
    };

    return <ul className="filetree-list">{renderNode(tree, 0)}</ul>;
};

export default FileTree;
