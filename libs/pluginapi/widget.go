package pluginapi

import "context"

// WidgetPlugin is a plugin that shows widgets on the home page of a workspace.
type WidgetPlugin interface {
	Plugin
	Widgets() ([]Widget, error)
}

// Widget is one card on the home page. The hub calls Load at each read, as the
// acting user in the acting workspace, and keeps no answer.
type Widget interface {
	Meta() WidgetMeta
	Load(ctx context.Context) (WidgetAnswer, error)
}

// WidgetKind names the shape of the data of a widget. The deck holds one
// component for each kind.
type WidgetKind string

// The kinds of a widget.
const (
	WidgetKindStat WidgetKind = "stat"
	WidgetKindList WidgetKind = "list"
)

// WidgetSize names the space of a widget on the home page.
type WidgetSize string

// The sizes of a widget, from one column and one row to four columns and two rows.
const (
	WidgetSizeXS WidgetSize = "xs"
	WidgetSizeS  WidgetSize = "s"
	WidgetSizeM  WidgetSize = "m"
	WidgetSizeL  WidgetSize = "l"
)

// WidgetState names what one answer of a widget says.
type WidgetState string

// The states of an answer.
const (
	WidgetStateReady    WidgetState = "ready"
	WidgetStateEmpty    WidgetState = "empty"
	WidgetStateNotSetUp WidgetState = "not_set_up"
)

// WidgetMeta declares a widget. Key is unique within the plugin, and Permission is
// a local name that the plugin declares, as for a route.
type WidgetMeta struct {
	Key        string
	Title      string
	Kind       WidgetKind
	Size       WidgetSize
	Permission string
}

// WidgetAnswer is one answer of a widget. Every text is plain, and every link is a
// path relative to the plugin. The answer helpers build each state.
type WidgetAnswer struct {
	State   WidgetState
	Message string
	Link    string
	Stat    *StatData
	List    *ListData
}

// StatData is the data of a stat widget. The plugin formats Value as a person reads it.
type StatData struct {
	Value   string
	Unit    string
	Caption string
	Link    string
}

// ListData is the data of a list widget. More links to the page that shows every item.
type ListData struct {
	Items []ListItem
	More  string
}

// ListItem is one line of a list widget.
type ListItem struct {
	Label string
	Meta  string
	Link  string
}

// FuncWidget binds a meta to a load function. NewWidget builds it.
type FuncWidget struct {
	meta WidgetMeta
	load func(ctx context.Context) (WidgetAnswer, error)
}

var _ Widget = (*FuncWidget)(nil)

// NewWidget builds a Widget from its meta and its load function.
func NewWidget(
	meta WidgetMeta,
	load func(ctx context.Context) (WidgetAnswer, error),
) *FuncWidget {
	return &FuncWidget{meta: meta, load: load}
}

// Meta describes the widget to the hub.
func (w *FuncWidget) Meta() WidgetMeta {
	return w.meta
}

// Load answers the data of the widget for the acting user.
func (w *FuncWidget) Load(ctx context.Context) (WidgetAnswer, error) {
	return w.load(ctx)
}

// StatAnswer answers the data of a stat widget.
func StatAnswer(data StatData) WidgetAnswer {
	return WidgetAnswer{State: WidgetStateReady, Stat: &data}
}

// ListAnswer answers the data of a list widget.
func ListAnswer(data ListData) WidgetAnswer {
	return WidgetAnswer{State: WidgetStateReady, List: &data}
}

// EmptyAnswer answers a widget that holds nothing to show, with the message that
// the deck shows in its place.
func EmptyAnswer(message string) WidgetAnswer {
	return WidgetAnswer{State: WidgetStateEmpty, Message: message}
}

// NotSetUpAnswer answers a widget that waits for the person to set something up,
// with the message and the link to the page that does it.
func NotSetUpAnswer(message, link string) WidgetAnswer {
	return WidgetAnswer{State: WidgetStateNotSetUp, Message: message, Link: link}
}
