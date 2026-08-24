package ui

import "charm.land/bubbles/v2/key"

// Every binding lives here, once, so the footer hints and the full help
// screen (opened with ?) render from the exact same source instead of two
// hand-maintained copies that can drift apart.
var (
	// Global — available whenever a text field doesn't own the keyboard.
	keyQuit     = key.NewBinding(key.WithKeys("q", "Q"), key.WithHelp("q", "quit"))
	keyHelp     = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help"))
	keyConnect  = key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "connection"))
	keySwitch   = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch panel"))
	keyUpload   = key.NewBinding(key.WithKeys("U"), key.WithHelp("U", "upload marked"))
	keyDownload = key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "download marked"))

	// File panels (Local, Remote).
	keyOpen     = key.NewBinding(key.WithKeys("enter", "l"), key.WithHelp("l/enter", "open dir"))
	keyUp       = key.NewBinding(key.WithKeys("-", "backspace", "h"), key.WithHelp("h/-", "go up"))
	keyMark     = key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "mark"))
	keyTransfer = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "transfer"))
	keyRefresh  = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh"))
	keySortNext = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort"))
	keySortFlip = key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "reverse sort"))

	// Connection dialog.
	keyNextField    = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field"))
	keyPrevField    = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev field"))
	keyProtocolPrev = key.NewBinding(key.WithKeys("left"))
	keyProtocolNext = key.NewBinding(key.WithKeys("right", "space"))
	keyProtocol     = key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "protocol")) // display only
	keySubmit       = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect"))

	// esc means something different per context (abandon a connection
	// attempt, close the dialog, close help), so matching uses one shared
	// binding while each context supplies its own help text below.
	keyEsc              = key.NewBinding(key.WithKeys("esc"))
	keyCancelConnecting = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "abandon"))
	keyCancelConnection = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close"))
	keyCancelHelp       = key.NewBinding(key.WithKeys("esc", "?"), key.WithHelp("esc", "close"))
)

// footerKeyMap drives the always-visible footer: only what's actionable
// from where the user is right now, never the full reference — that's ?.
type footerKeyMap struct {
	focus      focus
	connecting bool
	helpOpen   bool
}

func (k footerKeyMap) ShortHelp() []key.Binding {
	switch {
	case k.connecting:
		return []key.Binding{keyCancelConnecting}
	case k.helpOpen:
		return []key.Binding{keyCancelHelp}
	case k.focus == focusConnectionBar:
		return []key.Binding{keyNextField, keyPrevField, keyProtocol, keySubmit, keyCancelConnection}
	default:
		return []key.Binding{keyOpen, keyUp, keyMark, keyTransfer, keyHelp}
	}
}

func (footerKeyMap) FullHelp() [][]key.Binding { return nil }

// helpGroups is the complete reference the help screen renders, grouped by
// context; helpGroupTitles names each group in the same order.
var helpGroupTitles = []string{"Global", "File panels", "Connection dialog"}

func helpGroups() [][]key.Binding {
	return [][]key.Binding{
		{keyQuit, keyHelp, keyConnect, keySwitch, keyUpload, keyDownload},
		{keyOpen, keyUp, keyMark, keyTransfer, keyRefresh, keySortNext, keySortFlip},
		{keyNextField, keyPrevField, keyProtocol, keySubmit, keyCancelConnection},
	}
}
