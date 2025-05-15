package main

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/tgezginis/tesla-tracking-app/pkg/gui"
	"github.com/tgezginis/tesla-tracking-app/pkg/i18n"
	"github.com/tgezginis/tesla-tracking-app/pkg/tesla"
	"github.com/tgezginis/tesla-tracking-app/pkg/updater"
	"github.com/tgezginis/tesla-tracking-app/pkg/version"
)

func main() {
	// Initialize i18n
	i18n.Init()
	
	// Create the application
	a := app.New()
	a.Settings().SetTheme(theme.LightTheme())
	
	w := a.NewWindow(i18n.Text("app_title") + " v" + version.String())
	
	w.Resize(fyne.NewSize(1920, 1080))
	w.SetFixedSize(false)
	w.SetPadded(true)
	w.CenterOnScreen()
	
	w.SetMaster()
	
	teslaAuth := tesla.NewTeslaAuth()
	
	// Sol alt köşe için versiyon etiketi
	versionLabel := widget.NewLabelWithStyle(version.String(), fyne.TextAlignLeading, fyne.TextStyle{})
	
	// Function to set content with version label
	setContentWithVersion := func(screenContent fyne.CanvasObject) {
		versionedContent := container.NewBorder(
			nil,  // top
			container.NewHBox(versionLabel, layout.NewSpacer()), // bottom
			nil,  // left
			nil,  // right
			screenContent, // center
		)
		w.SetContent(versionedContent)
	}
	
	// Show loading screen
	loadingContent := container.NewCenter(
		container.NewVBox(
			widget.NewLabel(i18n.Text("app_title")),
			widget.NewProgressBarInfinite(),
			widget.NewLabel(i18n.Text("starting")),
		),
	)
	setContentWithVersion(loadingContent)
	w.Show()
	
	// Check for updates
	go checkForUpdates(w)
	
	var showAuthScreenFunc func()
	var showOrdersScreenFunc func()
	
	showOrdersScreenFunc = func() {
		ordersScreen := gui.NewOrdersScreen(a, w, teslaAuth, func() { // onLogout callback
			showAuthScreenFunc()
		})
		screenContent := ordersScreen.GetContent()
		setContentWithVersion(screenContent)
		ordersScreen.PerformInitialSetup() // Perform initial setup after content is set
	}
	
	showAuthScreenFunc = func() {
		authScreen := gui.NewAuthScreen(a, w, teslaAuth, func() { // onComplete callback
			showOrdersScreenFunc()
		})
		
		if err := teslaAuth.LoadTokensFromFile(); err == nil && teslaAuth.IsTokenValid() {
			log.Println("Token loaded and valid, showing orders screen.")
			showOrdersScreenFunc()
			return
		}
		log.Println("Token not found or invalid, showing auth screen.")
		screenContent := authScreen.GetContent()
		setContentWithVersion(screenContent)
		authScreen.PerformInitialSetup() // Perform initial setup after content is set
	}
	
	showAuthScreenFunc()
	
	a.Run()
}

// checkForUpdates checks if there's a new version and prompts the user to update
func checkForUpdates(w fyne.Window) {
	hasUpdate, release, err := updater.HasUpdate()
	if err != nil {
		log.Printf("Error checking for updates: %v", err)
		return
	}
	
	if !hasUpdate {
		log.Println("No updates available")
		return
	}
	
	// Show update dialog on UI thread
	updateMessage := fmt.Sprintf(i18n.Text("update_available"), release.Version())
	dialog.ShowConfirm(
		i18n.Text("update_title"),
		updateMessage,
		func(update bool) {
			if update {
				performUpdate(w, release)
			}
		},
		w,
	)
}

// performUpdate performs the actual update and informs the user
func performUpdate(w fyne.Window, release *selfupdate.Release) {
	progress := dialog.NewProgress(i18n.Text("updating"), i18n.Text("downloading_update"), w)
	progress.Show()
	
	// Start the update in a goroutine
	go func() {
		defer progress.Hide()
		err := updater.DoUpdate(release)
		
		if err != nil {
			// Show error dialog
			dialog.ShowError(fmt.Errorf(i18n.Text("update_error"), err), w)
			return
		}
		
		// Show success dialog and inform the user they need to restart
		dialog.ShowInformation(
			i18n.Text("update_success_title"),
			i18n.Text("update_success_message"),
			w,
		)
	}()
} 