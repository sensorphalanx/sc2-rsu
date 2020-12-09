package fynemd

import (
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

// NavigationDrawer ...
type NavigationDrawer struct {
	widget.BaseWidget

	OnDeselect func(NavigationItem) bool
	OnSelect   func(NavigationItem)

	items    []NavigationItem
	objects  []fyne.CanvasObject
	selected int

	image     *widget.Icon      // dup: objects[0]
	separator *widget.Separator // dup: objects[3]
	subtitle  *canvas.Text      // dup: objects[2]
	title     *canvas.Text      // dup: objects[1]
}

// NewNavigationDrawer ...
func NewNavigationDrawer(title, subtitle string, items ...NavigationItem) *NavigationDrawer {
	sub := NewScaledText(TextSizeBody2, subtitle)
	sub.Color = theme.DisabledTextColor()

	ret := &NavigationDrawer{
		items:     items,
		image:     widget.NewIcon(theme.CancelIcon()),
		separator: widget.NewSeparator(),
		subtitle:  sub,
		title:     NewScaledText(TextSizeHeading5, title),
	}
	ret.objects = []fyne.CanvasObject{
		ret.title,
		ret.subtitle,
		ret.separator,
	}

	ret.image.Hide()
	ret.ExtendBaseWidget(ret)

	return ret
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
//
// Implements: fyne.Widget
func (nav *NavigationDrawer) CreateRenderer() fyne.WidgetRenderer {
	return NavigationDrawerRenderer(nav)
}

// Select ...
func (nav *NavigationDrawer) Select(id int) {
	if nav.OnDeselect != nil {
		// they can keepfocus (example: unsaved changes)
		if !nav.OnDeselect(nav.items[nav.selected]) {
			return
		}
	}

	// ! 4+ hard-coded
	// ! (*widget.Button) hard-coded
	if b, ok := nav.objects[4+nav.selected].(*widget.Button); ok {
		b.Style = widget.DefaultButton
		b.Refresh()
	}

	nav.selected = id

	// ! 4+ hard-coded
	// ! (*widget.Button) hard-coded
	if b, ok := nav.objects[4+nav.selected].(*widget.Button); ok {
		b.Style = widget.PrimaryButton
		b.Refresh()
	}

	if nav.OnSelect != nil {
		nav.OnSelect(nav.items[nav.selected])
	}
}

// SetImage ...
func (nav *NavigationDrawer) SetImage(image fyne.Resource) {
	nav.image.SetResource(image)
	nav.image.Hidden = image == nil
	nav.Refresh()
}

// SetSubtitle ...
func (nav *NavigationDrawer) SetSubtitle(subtitle string) {
	nav.subtitle.Hidden = subtitle == ""
	nav.subtitle.Text = subtitle
	nav.Refresh()
}

// SetTitle ...
func (nav *NavigationDrawer) SetTitle(title string) {
	nav.title.Hidden = title == ""
	nav.title.Text = title
	nav.Refresh()
}