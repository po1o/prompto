package config

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/color"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/regex"
	"github.com/po1o/prompto/src/runtime"
	runjobs "github.com/po1o/prompto/src/runtime/jobs"
	"github.com/po1o/prompto/src/segments/options"
	"github.com/po1o/prompto/src/template"

	"go.yaml.in/yaml/v3"
	c "golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// SegmentStyle the style of segment, for more information, see the constants
type SegmentStyle string

func (s *SegmentStyle) resolve(context any) SegmentStyle {
	value, err := template.Render(string(*s), context)

	// default to Plain
	if err != nil || value == "" {
		return Plain
	}

	return SegmentStyle(value)
}

type Segment struct {
	writer                  SegmentWriter
	env                     runtime.Environment
	Options                 options.Map `yaml:"options,omitempty"`
	Cache                   *Cache      `yaml:"cache,omitempty"`
	Alias                   string      `yaml:"alias,omitempty"`
	styleCache              SegmentStyle
	name                    string
	LeadingDiamond          string         `yaml:"leading_diamond,omitempty"`
	TrailingDiamond         string         `yaml:"trailing_diamond,omitempty"`
	RenderPendingIcon       string         `yaml:"render_pending_icon,omitempty"`
	Template                string         `yaml:"template,omitempty"`
	Foreground              color.Ansi     `yaml:"foreground,omitempty"`
	TemplatesLogic          template.Logic `yaml:"templates_logic,omitempty"`
	PowerlineSymbol         string         `yaml:"powerline_symbol,omitempty"`
	Background              color.Ansi     `yaml:"background,omitempty"`
	Filler                  string         `yaml:"filler,omitempty"`
	Type                    SegmentType    `yaml:"type,omitempty"`
	Style                   SegmentStyle   `yaml:"style,omitempty"`
	LeadingPowerlineSymbol  string         `yaml:"leading_powerline_symbol,omitempty"`
	RenderPendingBackground color.Ansi     `yaml:"render_pending_background,omitempty"`
	ForegroundTemplates     template.List  `yaml:"foreground_templates,omitempty"`
	Tips                    []string       `yaml:"tips,omitempty"`
	BackgroundTemplates     template.List  `yaml:"background_templates,omitempty"`
	Templates               template.List  `yaml:"templates,omitempty"`
	ExcludeFolders          []string       `yaml:"exclude_folders,omitempty"`
	IncludeFolders          []string       `yaml:"include_folders,omitempty"`
	Needs                   []string       `yaml:"-"`
	Timeout                 int            `yaml:"timeout,omitempty"`
	MaxWidth                int            `yaml:"max_width,omitempty"`
	MinWidth                int            `yaml:"min_width,omitempty"`
	Duration                time.Duration  `yaml:"-"`
	NameLength              int            `yaml:"-"`
	Index                   int            `yaml:"index,omitempty"`
	Interactive             bool           `yaml:"interactive,omitempty"`
	Enabled                 bool           `yaml:"-"`
	Newline                 bool           `yaml:"newline,omitempty"`
	InvertPowerline         bool           `yaml:"invert_powerline,omitempty"`
	Force                   bool           `yaml:"force,omitempty"`
	restored                bool           `yaml:"-"`
	Toggled                 bool           `yaml:"toggled,omitempty"`
}

// segmentAlias avoids recursion during YAML unmarshaling.
type segmentAlias Segment

// Clone returns a copy of the segment with runtime-only state reset.
// This allows reusing immutable segment config safely across renders.
func (segment *Segment) Clone() *Segment {
	if segment == nil {
		return nil
	}

	cloned := *segment
	cloned.writer = nil
	cloned.env = nil
	cloned.styleCache = ""
	cloned.name = ""
	cloned.Needs = nil
	cloned.Duration = 0
	cloned.NameLength = 0
	cloned.Index = 0
	cloned.Enabled = false
	cloned.Force = false
	cloned.restored = false

	return &cloned
}

func (segment *Segment) UnmarshalYAML(node *yaml.Node) error {
	// Decode into a map to handle field renaming
	var raw map[string]any
	if err := node.Decode(&raw); err != nil {
		return err
	}

	// If 'properties' exists and 'options' doesn't, rename it
	if props, hasProps := raw["properties"]; hasProps {
		if _, hasOptions := raw["options"]; !hasOptions {
			raw["options"] = props
			delete(raw, "properties")
		}
	}

	// Re-encode and decode into the struct
	modifiedNode := &yaml.Node{}
	if err := modifiedNode.Encode(raw); err != nil {
		return err
	}

	return modifiedNode.Decode((*segmentAlias)(segment))
}

func (segment *Segment) Name() string {
	if len(segment.name) != 0 {
		return segment.name
	}

	name := segment.Alias
	if name == "" {
		name = c.Title(language.English).String(string(segment.Type))
	}

	segment.name = name
	return name
}

func (segment *Segment) Execute(env runtime.Environment) {
	// segment timings for debug purposes
	var start time.Time
	if env.Flags().Debug {
		start = time.Now()
		segment.NameLength = len(segment.Name())
		defer func() {
			segment.Duration = time.Since(start)
		}()
	}

	defer segment.evaluateNeeds()

	err := segment.MapSegmentWithWriter(env)
	if err != nil || !segment.shouldIncludeFolder() {
		return
	}

	log.Debugf("segment: %s", segment.Name())

	if segment.isToggled() {
		return
	}

	if segment.restoreCache() {
		return
	}

	if shouldHideForWidth(segment.env, segment.MinWidth, segment.MaxWidth) {
		return
	}

	defer func() {
		if segment.Enabled {
			template.Cache.AddSegmentData(segment.Name(), segment.writer)
		}
	}()

	// Create Job for this goroutine so child processes can be tracked and killed on timeout
	if err := runjobs.CreateJobForGoroutine(segment.Name()); err != nil {
		log.Errorf("failed to create job for goroutine (segment: %s): %v", segment.Name(), err)
	}

	segment.Enabled = segment.writer.Enabled()
}

func (segment *Segment) Render(index int, force bool) bool {
	if !segment.Enabled && !force {
		return false
	}

	if force {
		segment.Force = true
	}

	segment.writer.SetIndex(index)

	text := segment.string()
	segment.Enabled = segment.Force || len(strings.ReplaceAll(text, " ", "")) > 0

	if !segment.Enabled {
		template.Cache.RemoveSegmentData(segment.Name())
		return false
	}

	segment.SetText(text)
	segment.setCache()

	// We do this to make `.Text` available for a cross-segment reference in an extra prompt.
	template.Cache.AddSegmentData(segment.Name(), segment.writer)

	return true
}

func (segment *Segment) Text() string {
	return segment.writer.Text()
}

func (segment *Segment) SetText(text string) {
	segment.writer.SetText(text)
}

// CopyWriterStateFrom copies runtime writer data from another segment instance of the same type.
// The target segment must already have its writer initialized via MapSegmentWithWriter.
func (segment *Segment) CopyWriterStateFrom(source *Segment) error {
	if segment == nil || source == nil {
		return nil
	}

	if segment.writer == nil || source.writer == nil {
		return nil
	}

	data, err := json.Marshal(source.writer)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, segment.writer); err != nil {
		return err
	}

	segment.Enabled = source.Enabled
	return nil
}

// EnsureWriter initializes the segment writer only when needed.
// This preserves existing writer state during repaint flows.
func (segment *Segment) EnsureWriter(env runtime.Environment) error {
	if segment.writer != nil {
		segment.env = env
		return nil
	}

	return segment.MapSegmentWithWriter(env)
}

func (segment *Segment) ResolveForeground() color.Ansi {
	if len(segment.ForegroundTemplates) != 0 {
		match := segment.ForegroundTemplates.FirstMatch(segment.writer, segment.Foreground.String())
		segment.Foreground = color.Ansi(match)
	}

	return segment.Foreground
}

func (segment *Segment) ResolveBackground() color.Ansi {
	if len(segment.BackgroundTemplates) != 0 {
		match := segment.BackgroundTemplates.FirstMatch(segment.writer, segment.Background.String())
		segment.Background = color.Ansi(match)
	}

	return segment.Background
}

func (segment *Segment) ResolveStyle() SegmentStyle {
	if len(segment.styleCache) != 0 {
		return segment.styleCache
	}

	segment.styleCache = segment.Style.resolve(segment.writer)

	return segment.styleCache
}

func (segment *Segment) IsPowerline() bool {
	style := segment.ResolveStyle()
	return style == Powerline || style == Accordion
}

func (segment *Segment) HasEmptyDiamondAtEnd() bool {
	if segment.ResolveStyle() != Diamond {
		return false
	}

	return segment.TrailingDiamond == ""
}

func (segment *Segment) hasCache() bool {
	return segment.Cache != nil && !segment.Cache.Duration.IsEmpty()
}

func (segment *Segment) isToggled() bool {
	segmentName := segment.Alias
	if segmentName == "" {
		segmentName = string(segment.Type)
	}

	if segment.env != nil && segment.env.Flags() != nil && len(segment.env.Flags().SegmentToggles) > 0 {
		if segment.env.Flags().SegmentToggles[segmentName] {
			log.Debugf("segment toggled off: %s", segment.Name())
			return true
		}

		return false
	}

	togglesMap, OK := cache.Get[map[string]bool](cache.Session, cache.TOGGLECACHE)
	if !OK || len(togglesMap) == 0 {
		log.Debug("no toggles found")
		return false
	}

	if togglesMap[segmentName] {
		log.Debugf("segment toggled off: %s", segment.Name())
		return true
	}

	return false
}

func (segment *Segment) restoreCache() bool {
	if !segment.hasCache() {
		return false
	}

	key, store := segment.cacheKeyAndStore()
	data, OK := cache.Get[string](store, key)
	if !OK {
		log.Debugf("no cache found for segment: %s, key: %s", segment.Name(), key)
		return false
	}

	err := json.Unmarshal([]byte(data), &segment.writer)
	if err != nil {
		log.Error(err)
	}

	segment.Enabled = true
	template.Cache.AddSegmentData(segment.Name(), segment.writer)

	log.Debug("restored segment from cache: ", segment.Name())

	segment.restored = true

	return true
}

func (segment *Segment) setCache() {
	if segment.restored || !segment.hasCache() {
		return
	}

	data, err := json.Marshal(segment.writer)
	if err != nil {
		log.Error(err)
		return
	}

	// TODO: check if we can make segmentwriter a generic Type indicator
	// that way we can actually get the value straight from cache.Get
	// and marchalling is obsolete
	key, store := segment.cacheKeyAndStore()
	cache.Set(store, key, string(data), segment.Cache.Duration)
}

func (segment *Segment) cacheKeyAndStore() (string, cache.Store) {
	format := "segment_cache_%s"
	switch segment.Cache.Strategy {
	case Session:
		return fmt.Sprintf(format, segment.Name()), cache.Session
	case Device:
		return fmt.Sprintf(format, segment.Name()), cache.Device
	case Folder:
		fallthrough
	default:
		return fmt.Sprintf(format, strings.Join([]string{segment.Name(), segment.folderKey()}, "_")), cache.Device
	}
}

// DaemonCacheKey returns a cache key for daemon mode.
func (segment *Segment) DaemonCacheKey() string {
	format := "daemon_cache_%s"
	if segment.Cache == nil {
		return fmt.Sprintf(format, strings.Join([]string{segment.Name(), segment.FolderKey()}, "_"))
	}

	if segment.Cache.Strategy == Session {
		return fmt.Sprintf(format, segment.Name())
	}

	return fmt.Sprintf(format, strings.Join([]string{segment.Name(), segment.FolderKey()}, "_"))
}

func (segment *Segment) folderKey() (key string) {
	if segment.env == nil {
		return ""
	}

	key = segment.env.Pwd()

	defer func() {
		if recover() == nil {
			return
		}
		key = segment.env.Pwd()
	}()

	if segment.writer == nil {
		return key
	}

	cacheKey, ok := segment.writer.CacheKey()
	if !ok || len(cacheKey) == 0 {
		return key
	}

	return cacheKey
}

// FolderKey returns the legacy folder-scoped cache key for a segment.
func (segment *Segment) FolderKey() string {
	return segment.folderKey()
}

func (segment *Segment) string() string {
	result := segment.Templates.Resolve(segment.writer, "", segment.TemplatesLogic)
	if len(result) != 0 {
		return result
	}

	if segment.Template == "" {
		segment.Template = segment.writer.Template()
	}

	text, err := template.Render(segment.Template, segment.writer)
	if err != nil {
		return err.Error()
	}

	return text
}

func (segment *Segment) shouldIncludeFolder() bool {
	if segment.env == nil {
		return true
	}

	cwdIncluded := segment.cwdIncluded()
	cwdExcluded := segment.cwdExcluded()

	return cwdIncluded && !cwdExcluded
}

func (segment *Segment) cwdIncluded() bool {
	if len(segment.IncludeFolders) == 0 {
		return true
	}

	return segment.env.DirMatchesOneOf(segment.env.Pwd(), segment.IncludeFolders)
}

func (segment *Segment) cwdExcluded() bool {
	return segment.env.DirMatchesOneOf(segment.env.Pwd(), segment.ExcludeFolders)
}

func (segment *Segment) evaluateNeeds() {
	value := segment.Template

	if len(segment.ForegroundTemplates) != 0 {
		value += strings.Join(segment.ForegroundTemplates, "")
	}

	if len(segment.BackgroundTemplates) != 0 {
		value += strings.Join(segment.BackgroundTemplates, "")
	}

	if len(segment.Templates) != 0 {
		value += strings.Join(segment.Templates, "")
	}

	if !strings.Contains(value, ".Segments.") {
		return
	}

	matches := regex.FindAllNamedRegexMatch(`\.Segments\.(?P<NAME>[a-zA-Z0-9]+)`, value)
	for _, name := range matches {
		segmentName := name["NAME"]

		if len(name) == 0 || slices.Contains(segment.Needs, segmentName) {
			continue
		}

		segment.Needs = append(segment.Needs, segmentName)
	}
}

// GetPendingText computes the text to display for a segment in pending state.
func (segment *Segment) GetPendingText(cachedText string, cfg *Config) (enabled bool, text string, background color.Ansi) {
	pendingIcon := segment.getPendingIcon(cfg)
	if cachedText == "" {
		cachedText = "..."
	}

	return true, pendingIcon + cachedText, segment.getPendingBackground(cfg)
}

func (segment *Segment) getPendingIcon(cfg *Config) string {
	if segment.RenderPendingIcon != "" {
		return segment.RenderPendingIcon
	}

	if cfg != nil && cfg.RenderPendingIcon != "" {
		return cfg.RenderPendingIcon
	}

	return "\uf254 "
}

func (segment *Segment) getPendingBackground(cfg *Config) color.Ansi {
	if segment.RenderPendingBackground != "" {
		return segment.RenderPendingBackground
	}

	if cfg != nil {
		return cfg.RenderPendingBackground
	}

	return ""
}
