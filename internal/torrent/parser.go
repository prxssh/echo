package torrent

import "time"

// File represents a file in a multi-file torrent
type File struct {
	Length  int64    // File size in bytes
	MD5Hash string   // Optional MD5 checksum
	Path    []string // Path components (directories + filename)
}

// Info holds the "info" dictionary of the torrent, merged for single and multi
// file mode
type Info struct {
	Name        string     // Filename (single-file) or directory name (mult-file)
	Length      int64      // File length (single-file) or zero for multi-file
	Files       []File     // List of files (multi-file) or nil for single-file
	PieceLength int64      // Number of bytes in each piece
	Pieces      [][20]byte // Concatenated 20-byte SHA1 hashes
	Private     bool       // Whether the torrent is private
}

// Metainfo represents the top-level of .torrent file structure
type Metainfo struct {
	Announce     string     // Tracker URL
	AnnounceList [][]string // Optional list of tracker URLs
	CreationDate time.Time  // Optional creation date (zero value if absent)
	Comment      string     // Optional comment
	CreatedBy    string     // Optional creator identifier
	Encoding     string     // Optional text encoding
	Info         *Info      // Info dictionary
}
