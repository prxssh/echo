package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Buffer pool to reduce allocations
var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// PrettyHandlerOptions configures the PrettyHandler behavior.
type PrettyHandlerOptions struct {
	// SlogOpts are the standard slog handler options
	SlogOpts slog.HandlerOptions
	// UseColor enables/disables colored output (default: true)
	// Automatically disabled when output is not a TTY
	UseColor bool
	// ShowSource includes source file information in logs
	ShowSource bool
	// FullSource shows the full file path instead of just filename
	FullSource bool
	// CompactJSON renders JSON attributes in a single line
	CompactJSON bool
	// TimeFormat customizes timestamp format (default: RFC3339)
	TimeFormat string
	// LevelWidth ensures consistent level string width for alignment
	LevelWidth int
	// DisableTimestamp omits timestamps from output
	DisableTimestamp bool
	// FieldSeparator separates different log components (default: " | ")
	FieldSeparator string
	// MaxFieldLength truncates field values longer than this (0 = no limit)
	MaxFieldLength int
	// SortKeys sorts JSON keys alphabetically
	SortKeys bool
	// DisableHTMLEscape disables HTML escaping in JSON output
	DisableHTMLEscape bool
}

// DefaultOptions returns production-ready default options.
func DefaultOptions() PrettyHandlerOptions {
	return PrettyHandlerOptions{
		SlogOpts: slog.HandlerOptions{
			Level: slog.LevelInfo,
		},
		UseColor:          true,
		ShowSource:        false,
		FullSource:        false,
		CompactJSON:       false,
		TimeFormat:        time.RFC3339,
		LevelWidth:        7,
		DisableTimestamp:  false,
		FieldSeparator:    " | ",
		MaxFieldLength:    0,
		SortKeys:          false,
		DisableHTMLEscape: true,
	}
}

// PrettyHandler implements a colorful, human-readable log handler for slog.
type PrettyHandler struct {
	opts   PrettyHandlerOptions
	writer io.Writer
	mu     *sync.Mutex // Pointer for copyability
	groups []string
	attrs  []slog.Attr

	// Color functions cached based on options
	colorTime    func(...interface{}) string
	colorLevel   map[slog.Level]func(...interface{}) string
	colorMessage func(...interface{}) string
	colorSource  func(...interface{}) string
	colorFields  func(...interface{}) string
	colorError   func(...interface{}) string
}

// NewPrettyHandler creates a new PrettyHandler with the given writer and
// options.
func NewPrettyHandler(w io.Writer, opts *PrettyHandlerOptions) *PrettyHandler {
	if opts == nil {
		defaultOpts := DefaultOptions()
		opts = &defaultOpts
	}

	// Validate and set defaults
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.RFC3339
	}
	if opts.LevelWidth < 5 {
		opts.LevelWidth = 7
	}
	if opts.FieldSeparator == "" {
		opts.FieldSeparator = " | "
	}

	h := &PrettyHandler{
		opts:   *opts,
		writer: w,
		mu:     &sync.Mutex{},
		groups: make([]string, 0),
		attrs:  make([]slog.Attr, 0),
	}

	h.initColorFuncs()

	return h
}

// initColorFuncs initializes color formatting functions based on options.
func (h *PrettyHandler) initColorFuncs() {
	if !h.opts.UseColor {
		noColor := func(a ...interface{}) string { return fmt.Sprint(a...) }
		h.colorTime = noColor
		h.colorMessage = noColor
		h.colorSource = noColor
		h.colorFields = noColor
		h.colorError = noColor
		h.colorLevel = make(map[slog.Level]func(...interface{}) string)
		for _, level := range []slog.Level{
			slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError,
		} {
			h.colorLevel[level] = noColor
		}
		return
	}

	// Set up colored output
	h.colorTime = color.New(color.FgHiBlack).SprintFunc()
	h.colorMessage = color.New(color.FgCyan).SprintFunc()
	h.colorSource = color.New(color.FgHiBlack).SprintFunc()
	h.colorFields = color.New(color.FgWhite).SprintFunc()
	h.colorError = color.New(color.FgRed, color.Bold).SprintFunc()

	h.colorLevel = map[slog.Level]func(...interface{}) string{
		slog.LevelDebug: color.New(color.FgMagenta).SprintFunc(),
		slog.LevelInfo:  color.New(color.FgBlue).SprintFunc(),
		slog.LevelWarn:  color.New(color.FgYellow).SprintFunc(),
		slog.LevelError: color.New(color.FgRed).SprintFunc(),
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.SlogOpts.Level.Level()
}

// Handle formats and writes the log record.
func (h *PrettyHandler) Handle(ctx context.Context, r slog.Record) error {
	// Get buffer from pool
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	// Build the log line
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add timestamp
	if !h.opts.DisableTimestamp {
		timestamp := r.Time.Format(h.opts.TimeFormat)
		buf.WriteString(h.colorTime(timestamp))
		buf.WriteString(h.opts.FieldSeparator)
	}

	// Add level
	level := h.formatLevel(r.Level)
	buf.WriteString(level)
	buf.WriteString(h.opts.FieldSeparator)

	// Add source if enabled
	if h.opts.ShowSource {
		source := h.extractSource(r.PC)
		if source != "" {
			buf.WriteString(h.colorSource(source))
			buf.WriteString(h.opts.FieldSeparator)
		}
	}

	// Add message
	buf.WriteString(h.colorMessage(r.Message))

	// Collect and add attributes
	attrs := h.collectAttributes(r)
	if len(attrs) > 0 {
		buf.WriteString(h.opts.FieldSeparator)
		if err := h.formatAttributes(buf, attrs); err != nil {
			// Fallback to simple formatting on error
			buf.WriteString(
				fmt.Sprintf(
					"(error formatting attributes: %v)",
					err,
				),
			)
		}
	}

	// Write the complete line
	buf.WriteByte('\n')
	_, err := h.writer.Write(buf.Bytes())
	return err
}

// WithAttrs returns a new handler with additional attributes.
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Create new handler with copied state
	newHandler := &PrettyHandler{
		opts:   h.opts,
		writer: h.writer,
		mu:     &sync.Mutex{},
		groups: append([]string(nil), h.groups...),
		attrs:  append(append([]slog.Attr(nil), h.attrs...), attrs...),
	}
	newHandler.initColorFuncs()

	return newHandler
}

// WithGroup returns a new handler with the given group name.
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Create new handler with added group
	newHandler := &PrettyHandler{
		opts:   h.opts,
		writer: h.writer,
		mu:     &sync.Mutex{},
		groups: append(append([]string(nil), h.groups...), name),
		attrs:  append([]slog.Attr(nil), h.attrs...),
	}
	newHandler.initColorFuncs()

	return newHandler
}

// formatLevel formats the log level with appropriate styling.
func (h *PrettyHandler) formatLevel(level slog.Level) string {
	levelStr := strings.ToUpper(level.String())

	// Pad for alignment
	if h.opts.LevelWidth > 0 {
		levelStr = fmt.Sprintf("%-*s", h.opts.LevelWidth, levelStr)
	}

	// Apply color
	if colorFunc, ok := h.colorLevel[level]; ok {
		return colorFunc(levelStr)
	}

	// Handle custom levels
	if level > slog.LevelError {
		return h.colorError(levelStr)
	}
	return levelStr
}

// extractSource extracts source information from program counter.
func (h *PrettyHandler) extractSource(pc uintptr) string {
	if pc == 0 {
		return ""
	}

	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()

	if frame.Function == "" {
		return ""
	}

	file := frame.File
	if !h.opts.FullSource {
		file = filepath.Base(file)
	}

	// Format as "file:line" or "file:line:function" for verbose mode
	source := fmt.Sprintf("%s:%d", file, frame.Line)

	if h.opts.SlogOpts.AddSource {
		// Add function name for extra verbosity
		funcName := frame.Function
		if idx := strings.LastIndex(funcName, "."); idx >= 0 {
			funcName = funcName[idx+1:]
		}
		source = fmt.Sprintf("%s:%s", source, funcName)
	}

	return source
}

// collectAttributes collects all attributes including groups and handler attrs.
func (h *PrettyHandler) collectAttributes(
	r slog.Record,
) map[string]interface{} {
	attrs := make(map[string]interface{})

	// Add handler's pre-configured attributes
	current := attrs
	for _, group := range h.groups {
		nested := make(map[string]interface{})
		current[group] = nested
		current = nested
	}

	// Add handler attributes
	for _, attr := range h.attrs {
		h.addAttribute(current, attr)
	}

	// Add record attributes
	r.Attrs(func(attr slog.Attr) bool {
		h.addAttribute(current, attr)
		return true
	})

	// Clean up empty groups
	h.cleanEmptyGroups(attrs)

	return attrs
}

// addAttribute adds an attribute to the map, handling special cases.
func (h *PrettyHandler) addAttribute(
	attrs map[string]interface{},
	attr slog.Attr,
) {
	value := attr.Value.Resolve()

	// Handle groups
	if value.Kind() == slog.KindGroup {
		group := make(map[string]interface{})
		for _, groupAttr := range value.Group() {
			h.addAttribute(group, groupAttr)
		}
		if len(group) > 0 {
			attrs[attr.Key] = group
		}
		return
	}

	// Convert value to appropriate type
	var v interface{}
	switch value.Kind() {
	case slog.KindTime:
		v = value.Time().Format(h.opts.TimeFormat)
	case slog.KindDuration:
		v = value.Duration().String()
	case slog.KindAny:
		v = value.Any()
		// Truncate if needed
		if h.opts.MaxFieldLength > 0 {
			if str, ok := v.(string); ok &&
				len(str) > h.opts.MaxFieldLength {
				v = str[:h.opts.MaxFieldLength] + "..."
			}
		}
	default:
		v = value.Any()
	}

	attrs[attr.Key] = v
}

// cleanEmptyGroups removes empty nested groups from the attributes map.
func (h *PrettyHandler) cleanEmptyGroups(attrs map[string]interface{}) {
	for key, value := range attrs {
		if nested, ok := value.(map[string]interface{}); ok {
			h.cleanEmptyGroups(nested)
			if len(nested) == 0 {
				delete(attrs, key)
			}
		}
	}
}

// formatAttributes formats attributes as JSON.
func (h *PrettyHandler) formatAttributes(
	buf *bytes.Buffer,
	attrs map[string]interface{},
) error {
	if len(attrs) == 0 {
		return nil
	}

	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(!h.opts.DisableHTMLEscape)

	if h.opts.CompactJSON {
		encoder.SetIndent("", "")
	} else {
		encoder.SetIndent("", "  ")
	}

	// Encode to temporary buffer first to apply coloring
	var jsonBuf bytes.Buffer
	encoder = json.NewEncoder(&jsonBuf)
	encoder.SetEscapeHTML(!h.opts.DisableHTMLEscape)
	if h.opts.CompactJSON {
		encoder.SetIndent("", "")
	} else {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(attrs); err != nil {
		return err
	}

	// Remove trailing newline from JSON encoder
	result := bytes.TrimRight(jsonBuf.Bytes(), "\n")

	// Apply color and write
	buf.WriteString(h.colorFields(string(result)))

	return nil
}
