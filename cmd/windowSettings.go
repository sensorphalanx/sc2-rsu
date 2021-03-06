package cmd

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"

	"github.com/spf13/viper"

	"github.com/AlbinoGeek/sc2-rsu/cmd/gui"
	"github.com/AlbinoGeek/sc2-rsu/sc2replaystats"
	"github.com/AlbinoGeek/sc2-rsu/sc2utils"
)

type windowSettings struct {
	*gui.WindowBase

	// do we have unsaved changes in the form?
	unsaved bool

	// widgets
	apiKey       *widget.Entry
	autoDownload *widget.Check
	checkUpdates *widget.Check
	replaysRoot  *widget.Entry
	updatePeriod *widget.Entry
}

// TODO: candidate for refactor
func (settings *windowSettings) Init() {
	w := settings.App.NewWindow("Settings")
	settings.SetWindow(w)

	settings.apiKey = widget.NewEntry()
	settings.apiKey.SetText(viper.GetString("apiKey"))
	settings.apiKey.Validator = func(key string) (err error) {
		if !sc2replaystats.ValidAPIKey(key) {
			err = errors.New("invalid API key format")
		}

		return
	}
	settings.apiKey.OnChanged = func(string) {
		settings.unsaved = true
	}

	settings.autoDownload = widget.NewCheck("Automatically Download Updates?", func(checked bool) {
		settings.unsaved = true
	})
	settings.autoDownload.SetChecked(viper.GetBool("update.automatic.enabled"))

	settings.updatePeriod = widget.NewEntry()
	settings.updatePeriod.SetText(getUpdateDuration().String())
	settings.updatePeriod.Validator = func(period string) (err error) {
		_, err = time.ParseDuration(period)
		return
	}
	settings.updatePeriod.OnChanged = func(string) {
		settings.unsaved = true
	}

	settings.checkUpdates = widget.NewCheck("Check for Updates Periodically?", func(checked bool) {
		settings.unsaved = true
		if checked {
			settings.autoDownload.Enable()
			settings.updatePeriod.Enable()
		} else {
			settings.autoDownload.Disable()
			settings.updatePeriod.Disable()
		}
	})
	settings.checkUpdates.SetChecked(viper.GetBool("update.check.enabled"))

	if !settings.checkUpdates.Checked {
		settings.autoDownload.Disable()
		settings.updatePeriod.Disable()
	}

	settings.unsaved = false // otherwise set by the above line

	settings.replaysRoot = widget.NewEntry()
	settings.replaysRoot.SetText(viper.GetString("replaysRoot"))
	settings.replaysRoot.OnChanged = func(string) {
		settings.unsaved = true
	}

	replaysRootSection := widget.NewVBox(
		fyne.NewContainerWithLayout(
			layout.NewFormLayout(),
			widget.NewLabel("Replays Root"),
			widget.NewHScrollContainer(settings.replaysRoot),
		),
		fyne.NewContainerWithLayout(
			layout.NewGridLayout(2),
			widget.NewButtonWithIcon("Find it for me...", theme.SearchIcon(), func() { go settings.findReplaysRoot() }),
			widget.NewButtonWithIcon("Browse...", theme.FolderOpenIcon(), func() {
				dlg := dialog.NewFolderOpen(settings.browseReplaysRoot, w)
				dlg.Resize(w.Canvas().Size().Subtract(fyne.NewSize(20, 20))) // ! can't be larger than the settings window
				dlg.Show()
			}),
		),
	)

	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(5, 5))

	btnSave := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), settings.save)
	btnSave.Importance = widget.HighImportance

	w.SetContent(widget.NewVBox(
		widget.NewCard(fmt.Sprintf("%s Settings", PROGRAM), "", widget.NewVBox(
			settings.checkUpdates,
			settings.autoDownload,
			fyne.NewContainerWithLayout(
				layout.NewFormLayout(),
				widget.NewLabel("Check Every"),
				settings.updatePeriod,
			),
		)),
		spacer,
		widget.NewCard("sc2ReplayStats Account", "", widget.NewVBox(
			fyne.NewContainerWithLayout(
				layout.NewFormLayout(),
				widget.NewLabel("API Key"),
				widget.NewHScrollContainer(settings.apiKey),
			),
			widget.NewButtonWithIcon("Login and Generate it for me...", theme.ComputerIcon(), settings.openLogin),
		)),
		spacer,
		widget.NewCard("StarCraft II", "", replaysRootSection),
		spacer,
		layout.NewSpacer(),
		widget.NewSeparator(),
		fyne.NewContainerWithLayout(
			layout.NewGridLayout(2),
			widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
				settings.onClose()
			}),
			btnSave,
		),
	))

	w.SetCloseIntercept(settings.onClose)
	w.SetOnClosed(func() {
		settings.SetWindow(nil)
	})

	w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyEscape {
			settings.onClose()
		}
	})

	w.SetPadded(false)
	w.SetFixedSize(true)

	w.Resize(fyne.NewSize(600, 600))
	w.CenterOnScreen()
	w.Show()
}

// TODO: candidate for refactor
func (settings *windowSettings) browseReplaysRoot(uri fyne.ListableURI, err error) {
	if err != nil {
		dialog.ShowError(err, settings.GetWindow())
		return
	}

	if uri == nil {
		return // cancelled
	}

	root := strings.TrimPrefix(uri.String(), "file://")

	// TODO: record the newly found accounts if confirmed
	settings.confirmValidReplaysRoot(root, func() {
		settings.unsaved = true
		settings.replaysRoot.SetText(root)
	})
}

// confirmValidReplaysRoot checks whether there are any accounts found at a
// given root, and if not, asks the user if they would like to use this root
// regardless. If they confirm, or accounts were found, callback is called.
func (settings *windowSettings) confirmValidReplaysRoot(root string, callback func()) {
	if accs, err := sc2utils.EnumerateAccounts(root); err == nil && len(accs) > 0 {
		callback()
		return
	}

	dialog.ShowConfirm("Invalid Directory!",
		fmt.Sprintf("We could not find any accounts in that directory.\nAre you sure you want to use it anyways?\n\n%s", root),
		func(ok bool) {
			if ok {
				callback()
			}
		}, settings.GetWindow())
}

// TODO: candidate for refactor
func (settings *windowSettings) findReplaysRoot() {
	w := settings.GetWindow()
	scanRoot := "/"

	if home, err := os.UserHomeDir(); err == nil {
		scanRoot = home
	}

	dlg := dialog.NewProgressInfinite("Searching for Replays Root...",
		"Please wait while we search for a valid Replays folder.\nThis could take several minutes.", w)
	dlg.Show()

	roots, err := sc2utils.FindReplaysRoot(scanRoot)

	dlg.Hide()

	if err != nil {
		dialog.ShowError(err, w)
		return
	}

	if len(roots) == 0 {
		dialog.ShowError(errors.New("no replay directories found"), w)
		return
	}

	if len(roots) == 1 {
		settings.confirmValidReplaysRoot(roots[0], func() {
			settings.unsaved = true
			settings.replaysRoot.SetText(roots[0])
			dialog.ShowInformation("Replays Root Found!", "We found your replays directory!", w)
		})

		return
	}

	selected := -1
	longest := ""

	for _, s := range roots {
		if l := len(s); l > len(longest) {
			longest = s
		}
	}

	listWidget := widget.NewList(func() int {
		return len(roots)
	}, func() fyne.CanvasObject {
		return widget.NewLabel(longest)
	}, func(id int, obj fyne.CanvasObject) {
		obj.(*widget.Label).SetText(roots[id])
	})
	listWidget.OnSelected = func(id int) {
		selected = id
	}
	dlg2 := dialog.NewCustomConfirm("Multiple Possible Roots Found",
		"Select", "Cancel", widget.NewHScrollContainer(listWidget), func(ok bool) {
			if !ok {
				return
			}

			settings.confirmValidReplaysRoot(roots[selected], func() {
				settings.unsaved = true
				settings.replaysRoot.SetText(roots[selected])
			})
		}, settings.GetWindow())

	size := fyne.MeasureText(longest, theme.TextSize(), fyne.TextStyle{})
	size.Height *= len(roots)

	dlg2.Resize(fyne.NewSize(60, 144).Add(size))
	dlg2.Show()
}

func (settings *windowSettings) onClose() {
	w := settings.GetWindow()

	if !settings.unsaved {
		w.Close()
		return
	}

	dialog.ShowConfirm("Unsaved Changes",
		"You have not saved your settings.\nAre you sure you want to discard amy changes?",
		func(ok bool) {
			if ok {
				w.Close()
			}
		}, w)
}

func (settings *windowSettings) openLogin() {
	w := settings.GetWindow()
	user := widget.NewEntry()
	pass := widget.NewPasswordEntry()

	// TODO: actually write a different warning for the gui instead of recycling the cli warning
	warning := widget.NewLabel(loginWarning[:strings.LastIndexByte(loginWarning[:len(loginWarning)-1], '.')+1])
	warning.Wrapping = fyne.TextWrapWord
	vbox := widget.NewVBox(
		warning,
		layout.NewSpacer(),
		widget.NewForm(
			widget.NewFormItem("Email", user),
			widget.NewFormItem("Password", pass),
		),
		layout.NewSpacer(),
	)

	dlg := dialog.NewCustomConfirm("Login to sc2replaystats", "Login", "Cancel", vbox, func(ok bool) {
		if ok {
			dlg2 := dialog.NewProgressInfinite("1) Login", "Setting up our login browser...", w)
			dlg2.Show()
			pw, browser, page, err := newBrowser()

			if pw != nil {
				defer pw.Stop()
			}

			if browser != nil {
				defer browser.Close()
			}

			if page != nil {
				defer page.Close()
			}

			dlg2.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("failed setting up browser: %v", err), w)
				return
			}

			dlg2 = dialog.NewProgressInfinite("2) Login", "Please wait while we try to login to sc2replaystats...", w)
			dlg2.Show()
			accid, err := login(page, user.Text, pass.Text)
			dlg2.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("login error: %v", err), w)
				return
			}

			dlg2 = dialog.NewProgressInfinite("3) Login", "Finding or Generating API Key...", w)
			dlg2.Show()
			key, err := extractAPIKey(page, accid)
			dlg2.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to get API key: %v", err), w)
				return
			}

			settings.apiKey.SetText(key)
			settings.apiKey.Validate()
		}
	}, w)

	vbox.Resize(fyne.NewSize(999, 280))
	dlg.Resize(fyne.NewSize(420, 280))
	dlg.Show()
}

func (settings *windowSettings) save() {
	w := settings.GetWindow()

	if err := settings.validate(); err != nil {
		dialog.ShowError(err, w)
		return
	}

	main := settings.UI.Windows[WindowMain].(*windowMain)
	if main.gettingStarted == 3 && settings.apiKey.Text != "" {
		main.openGettingStarted4()
	}

	if main.gettingStarted == 2 && settings.replaysRoot.Text != "" {
		main.openGettingStarted3()
	}

	var changes bool

	if oldKey := viper.Get("apikey"); oldKey != settings.apiKey.Text {
		viper.Set("apikey", settings.apiKey.Text)

		changes = true

		// Use the new apiKey immediately
		sc2api = sc2replaystats.New(settings.apiKey.Text)
	}

	if oldRoot := viper.Get("replaysRoot"); oldRoot != settings.replaysRoot.Text {
		viper.Set("replaysRoot", settings.replaysRoot.Text)

		changes = true
	}

	if changes {
		main.genAccountList()
		main.setupUploader()
	}

	viper.Set("update.automatic.enabled", settings.autoDownload.Checked)
	viper.Set("update.check.enabled", settings.checkUpdates.Checked)
	viper.Set("version", VERSION)

	if err := saveConfig(); err != nil {
		dialog.ShowError(err, w)

		return
	}

	dialog.ShowInformation("Saved!", "Your settings have been saved.", settings.UI.Windows[WindowMain].GetWindow())
	settings.unsaved = false

	w.Close()
}

func (settings *windowSettings) validate() error {
	if err := settings.apiKey.Validate(); settings.apiKey.Text != "" && err != nil {
		return fmt.Errorf("invalid value for \"API Key\": %v", err)
	}

	if err := settings.replaysRoot.Validate(); err != nil {
		return fmt.Errorf("invalid value for \"Replays Root\": %v", err)
	}

	if err := settings.updatePeriod.Validate(); err != nil {
		return fmt.Errorf("invalid value for \"Check Every\": %v", err)
	}

	return nil
}
