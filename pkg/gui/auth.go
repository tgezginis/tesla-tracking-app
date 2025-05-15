package gui

import (
	"fmt"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/tgezginis/tesla-tracking-app/pkg/i18n"
	"github.com/tgezginis/tesla-tracking-app/pkg/tesla"
	"github.com/tgezginis/tesla-tracking-app/pkg/utils"
)

type AuthScreen struct {
	window         fyne.Window
	app            fyne.App
	authURL        string
	teslaAuth      *tesla.TeslaAuth
	onComplete     func()
	
	headerLabel    *widget.Label
	authDescLabel  *widget.Label
	openButton     *widget.Button
	submitButton   *widget.Button 
	urlLabel       *widget.Label
	langSelect     *widget.Select
}

func NewAuthScreen(app fyne.App, window fyne.Window, teslaAuth *tesla.TeslaAuth, onComplete func()) *AuthScreen {
	return &AuthScreen{
		app:        app,
		window:     window,
		teslaAuth:  teslaAuth,
		onComplete: onComplete,
	}
}

func (s *AuthScreen) Show() {
	s.headerLabel = widget.NewLabelWithStyle(i18n.Text("auth_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	header := container.NewHBox(s.headerLabel)
	
	content := container.NewVBox()
	
	s.langSelect = s.createLanguageSelector()
	header.Add(layout.NewSpacer())
	header.Add(s.langSelect)
	
	mainContainer := container.NewPadded(
		container.NewBorder(
			header,
			nil, nil, nil,
			container.NewPadded(content),
		),
	)
	
	if err := s.teslaAuth.LoadTokensFromFile(); err == nil && s.teslaAuth.IsTokenValid() {
		s.onComplete()
		return
	} else {
		s.window.SetContent(mainContainer)
		s.showAuthForm()
	}
}

func (s *AuthScreen) createLanguageSelector() *widget.Select {
	langOptions := []string{i18n.Text("english"), i18n.Text("turkish")}
	langSelect := widget.NewSelect(langOptions, func(selected string) {
		prevLang := i18n.CurrentLang
		
		if selected == i18n.Text("english") {
			i18n.SetLanguage(i18n.LangEnglish)
		} else if selected == i18n.Text("turkish") {
			i18n.SetLanguage(i18n.LangTurkish)
		}
		
		if prevLang != i18n.CurrentLang {
			s.window.SetTitle(i18n.Text("app_title"))
			
			s.updateLanguageUI()
		}
	})
	
	if i18n.CurrentLang == i18n.LangEnglish {
		langSelect.SetSelected(i18n.Text("english"))
	} else {
		langSelect.SetSelected(i18n.Text("turkish"))
	}
	
	return langSelect
}

func (s *AuthScreen) updateLanguageUI() {
	fyne.Do(func() {
		s.window.SetTitle(i18n.Text("app_title"))

		if s.headerLabel != nil {
			s.headerLabel.SetText(i18n.Text("auth_title"))
			s.headerLabel.Refresh()
		}

		if s.langSelect != nil {
			currentSelection := s.langSelect.Selected
			s.langSelect.Options = []string{i18n.Text("english"), i18n.Text("turkish")}
			// Try to preserve selection or default
			newSelection := i18n.Text("english") // Default
			if i18n.CurrentLang == i18n.LangTurkish {
				newSelection = i18n.Text("turkish")
			}
			// If the old selection text matches a new option text, keep it.
			// This might not be perfect if keys are different from displayed values after translation.
			for _, opt := range s.langSelect.Options {
				if opt == currentSelection {
					newSelection = currentSelection
					break
				}
			}
			s.langSelect.SetSelected(newSelection)
			s.langSelect.Refresh()
		}

		if s.authDescLabel != nil {
			s.authDescLabel.SetText(i18n.Text("auth_description"))
			s.authDescLabel.Refresh()
		}

		if s.openButton != nil {
			s.openButton.SetText(i18n.Text("login"))
			s.openButton.Refresh()
		}

		if s.submitButton != nil {
			s.submitButton.SetText(i18n.Text("login"))
			s.submitButton.Refresh()
		}

		if s.urlLabel != nil {
			// Assuming s.urlLabel is for the redirect URL info which might change based on language
			s.urlLabel.SetText(i18n.Text("redirect_url_info")) 
			s.urlLabel.Refresh()
		}
	})
}

func (s *AuthScreen) showAuthForm() {
	s.authURL = s.teslaAuth.GetAuthURL()
	
	s.headerLabel = widget.NewLabelWithStyle(i18n.Text("auth_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	header := container.NewHBox(s.headerLabel)
	
	s.langSelect = s.createLanguageSelector()
	header.Add(layout.NewSpacer())
	header.Add(s.langSelect)
	
	s.authDescLabel = widget.NewLabelWithStyle(i18n.Text("auth_description"), fyne.TextAlignCenter, fyne.TextStyle{})
	content := container.NewVBox(s.authDescLabel)
	
	s.openButton = widget.NewButton(i18n.Text("login"), func() {
		if err := utils.OpenBrowser(s.authURL); err != nil {
			dialog.ShowError(fmt.Errorf(i18n.Text("error_opening_browser"), err), s.window)
		}
	})
	s.openButton.Importance = widget.HighImportance
	
	entry := widget.NewEntry()
	entry.SetPlaceHolder(i18n.Text("auth_description"))
	
	s.submitButton = widget.NewButton(i18n.Text("login"), func() {
		if entry.Text == "" {
			dialog.ShowError(fmt.Errorf(i18n.Text("login_error")), s.window)
			return
		}
		
		code, err := extractAuthCode(entry.Text)
		if err != nil {
			alternativeCode := extractAuthCodeAlternative(entry.Text)
			if alternativeCode != "" {
				code = alternativeCode
			} else {
				dialog.ShowError(err, s.window)
				return
			}
		}
		
		progress := dialog.NewProgressInfinite(i18n.Text("login_progress"), i18n.Text("loading"), s.window)
		progress.Show()
		
		go func() {
			defer progress.Hide()
			if err := s.teslaAuth.ExchangeCodeForTokens(code); err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   i18n.Text("error"),
					Content: fmt.Sprintf("%s %v", i18n.Text("login_error"), err),
				})
				dialog.ShowError(fmt.Errorf("%s %v", i18n.Text("login_error"), err), s.window)
				return
			}
			
			if err := s.teslaAuth.SaveTokensToFile(); err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   i18n.Text("error"),
					Content: fmt.Sprintf("%s %v", i18n.Text("login_error"), err),
				})
				dialog.ShowError(fmt.Errorf("%s %v", i18n.Text("login_error"), err), s.window)
				return
			}
			
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   i18n.Text("login_success"),
				Content: i18n.Text("login_success"),
			})
			
			dialog.ShowInformation(i18n.Text("login_success"), i18n.Text("login_success"), s.window)
			s.onComplete()
		}()
	})
	s.submitButton.Importance = widget.HighImportance
	
	buttonArea := container.NewHBox(layout.NewSpacer(), s.openButton, layout.NewSpacer())
	
	s.urlLabel = widget.NewLabelWithStyle(i18n.Text("redirect_url_info"), fyne.TextAlignCenter, fyne.TextStyle{})
	
	submitArea := container.NewHBox(layout.NewSpacer(), s.submitButton, layout.NewSpacer())
	
	content.Add(container.NewPadded(widget.NewLabel("")))
	content.Add(buttonArea)
	content.Add(container.NewPadded(widget.NewLabel("")))
	content.Add(s.urlLabel)
	content.Add(container.NewPadded(entry))
	content.Add(container.NewPadded(widget.NewLabel("")))
	content.Add(submitArea)
	
	mainContainer := container.NewPadded(
		container.NewBorder(
			header,
			nil, nil, nil,
			container.NewPadded(content),
		),
	)
	
	fyne.Do(func() {
		s.window.SetContent(mainContainer)
	})
}

func extractAuthCode(url string) (string, error) {
	if !strings.Contains(url, "code=") {
		return "", fmt.Errorf(i18n.Text("invalid_url"))
	}
	
	parts := strings.Split(url, "code=")
	if len(parts) < 2 {
		return "", fmt.Errorf(i18n.Text("invalid_url_format"))
	}
	
	code := parts[1]
	if idx := strings.Index(code, "&"); idx != -1 {
		code = code[:idx]
	}
	
	return code, nil
}

func extractAuthCodeAlternative(urlStr string) string {
	if !strings.HasPrefix(urlStr, "http") {
		urlStr = "https://" + urlStr
	}
	
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	
	queryParams, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return ""
	}
	
	code := queryParams.Get("code")
	
	return code
}

func (s *AuthScreen) GetContent() fyne.CanvasObject {
	s.headerLabel = widget.NewLabelWithStyle(i18n.Text("auth_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	header := container.NewHBox(s.headerLabel)
	
	s.langSelect = s.createLanguageSelector()
	header.Add(layout.NewSpacer())
	header.Add(s.langSelect)

	s.authURL = s.teslaAuth.GetAuthURL()
	
	s.authDescLabel = widget.NewLabelWithStyle(i18n.Text("auth_description"), fyne.TextAlignCenter, fyne.TextStyle{})
	contentVBox := container.NewVBox(s.authDescLabel)
	
	s.openButton = widget.NewButton(i18n.Text("login"), func() {
		if err := utils.OpenBrowser(s.authURL); err != nil {
			dialog.ShowError(fmt.Errorf(i18n.Text("error_opening_browser"), err), s.window)
		}
	})
	s.openButton.Importance = widget.HighImportance
	
	entry := widget.NewEntry()
	entry.SetPlaceHolder(i18n.Text("auth_description"))
	
	s.submitButton = widget.NewButton(i18n.Text("login"), func() {
		if entry.Text == "" {
			dialog.ShowError(fmt.Errorf(i18n.Text("login_error")), s.window)
			return
		}
		
		code, err := extractAuthCode(entry.Text)
		if err != nil {
			alternativeCode := extractAuthCodeAlternative(entry.Text)
			if alternativeCode != "" {
				code = alternativeCode
			} else {
				dialog.ShowError(err, s.window)
				return
			}
		}
		
		progress := dialog.NewProgressInfinite(i18n.Text("login_progress"), i18n.Text("loading"), s.window)
		progress.Show()
		
		go func() {
			defer progress.Hide()
			if err := s.teslaAuth.ExchangeCodeForTokens(code); err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   i18n.Text("error"),
					Content: fmt.Sprintf("%s %v", i18n.Text("login_error"), err),
				})
				dialog.ShowError(fmt.Errorf("%s %v", i18n.Text("login_error"), err), s.window)
				return
			}
			
			if err := s.teslaAuth.SaveTokensToFile(); err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   i18n.Text("error"),
					Content: fmt.Sprintf("%s %v", i18n.Text("login_error"), err),
				})
				dialog.ShowError(fmt.Errorf("%s %v", i18n.Text("login_error"), err), s.window)
				return
			}
			
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   i18n.Text("login_success"),
				Content: i18n.Text("login_success"),
			})
			
			dialog.ShowInformation(i18n.Text("login_success"), i18n.Text("login_success"), s.window)
			s.onComplete()
		}()
	})
	s.submitButton.Importance = widget.HighImportance
	
	buttonArea := container.NewHBox(layout.NewSpacer(), s.openButton, layout.NewSpacer())
	
	s.urlLabel = widget.NewLabelWithStyle(i18n.Text("redirect_url_info"), fyne.TextAlignCenter, fyne.TextStyle{})
	
	submitArea := container.NewHBox(layout.NewSpacer(), s.submitButton, layout.NewSpacer())
	
	contentVBox.Add(container.NewPadded(widget.NewLabel("")))
	contentVBox.Add(buttonArea)
	contentVBox.Add(container.NewPadded(widget.NewLabel("")))
	contentVBox.Add(s.urlLabel)
	contentVBox.Add(container.NewPadded(entry))
	contentVBox.Add(container.NewPadded(widget.NewLabel("")))
	contentVBox.Add(submitArea)
	
	mainContainer := container.NewPadded(
		container.NewBorder(
			header,
			nil, nil, nil,
			container.NewPadded(contentVBox),
		),
	)
	return mainContainer
}

func (s *AuthScreen) PerformInitialSetup() {
	s.updateLanguageUI()
} 