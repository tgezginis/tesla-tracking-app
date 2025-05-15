package gui

import (
	"fmt"
	"image/color"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tgezginis/tesla-tracking-app/pkg/i18n"
	"github.com/tgezginis/tesla-tracking-app/pkg/tesla"
	"github.com/tgezginis/tesla-tracking-app/pkg/utils"
)


type OrdersScreen struct {
	window           fyne.Window
	app              fyne.App
	teslaAuth        *tesla.TeslaAuth
	orderManager     *tesla.OrderManager
	ordersList       *widget.List
	detailsContainer *fyne.Container
	orders           []tesla.DetailedOrder
	refreshTimer     *time.Timer
	refreshInterval  time.Duration
	isAutoRefresh    bool
	refreshSelect    *widget.Select
	lastRefreshTime  time.Time
	refreshStatusLabel *widget.Label
	changedFields    map[string]bool
	onLogout         func() 
	
	
	titleLabel      *widget.Label
	logoutButton    *widget.Button
	refreshButton   *widget.Button
	autoRefreshLabel *widget.Label
	langSelect      *widget.Select
	noOrdersLabel   *widget.Label
	
	
	mainDetailTitle *widget.Label
	orderTitles     []*canvas.Text
	currentOrderDetail tesla.DetailedOrder
}


func NewOrdersScreen(app fyne.App, window fyne.Window, teslaAuth *tesla.TeslaAuth, onLogout func()) *OrdersScreen {
	orderManager := tesla.NewOrderManager(teslaAuth)
	
	return &OrdersScreen{
		app:             app,
		window:          window,
		teslaAuth:       teslaAuth,
		orderManager:    orderManager,
		refreshInterval: time.Minute * 5, 
		isAutoRefresh:   false,
		lastRefreshTime: time.Now(),
		changedFields:   make(map[string]bool),
		onLogout:        onLogout,
	}
}


func (s *OrdersScreen) Show() {
	content := s.GetContent()
	
	fyne.Do(func() {
		s.window.SetContent(content)
		s.window.SetTitle(i18n.Text("app_title"))
	})
	
	s.fetchOrders()
	
	s.window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyEscape && s.refreshTimer != nil {
			s.refreshTimer.Stop()
		}
	})
}


func (s *OrdersScreen) GetContent() fyne.CanvasObject {
	s.noOrdersLabel = widget.NewLabelWithStyle(i18n.Text("select_order"), fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	s.detailsContainer = container.NewPadded(
		container.NewVBox(
			s.noOrdersLabel,
		),
	)
	
	s.refreshStatusLabel = widget.NewLabel(i18n.Text("last_refresh") + ": " + i18n.Text("not_refreshed"))
	
	s.refreshSelect = widget.NewSelect(
		[]string{i18n.Text("off"), "5 " + i18n.Text("minutes"), "10 " + i18n.Text("minutes"), 
			"15 " + i18n.Text("minutes"), "30 " + i18n.Text("minutes"), "60 " + i18n.Text("minutes")},
		func(selected string) {
			s.handleRefreshIntervalChange(selected)
		})
	s.refreshSelect.SetSelected("5 " + i18n.Text("minutes")) // Default refresh interval
	
	s.langSelect = s.createLanguageSelector()
	
	s.refreshButton = widget.NewButton(i18n.Text("refresh"), func() {
		fyne.Do(func() { // Ensure UI updates are on the main thread
			s.fetchOrders()
		})
	})
	s.refreshButton.Importance = widget.HighImportance
	
	s.logoutButton = widget.NewButton(i18n.Text("logout"), func() {
		dialog.ShowConfirm(
			i18n.Text("logout_confirmation"), 
			i18n.Text("logout_confirmation_message"),
			func(confirmed bool) {
				if confirmed {
					if s.refreshTimer != nil {
						s.refreshTimer.Stop()
					}
					if err := os.Remove(tesla.TokenFile); err != nil {
						fmt.Printf("Error removing token file: %v\n", err)
					}
					s.onLogout()
				}
			},
			s.window,
		)
	})
	
	s.autoRefreshLabel = widget.NewLabel(i18n.Text("auto_refresh") + ":")
	refreshControls := container.NewHBox(
		s.autoRefreshLabel,
		s.refreshSelect,
		layout.NewSpacer(),
		s.refreshStatusLabel,
		layout.NewSpacer(),
		s.refreshButton,
	)
	
	langControls := container.NewHBox(
		widget.NewLabel(i18n.Text("language") + ":"),
		s.langSelect,
	)
	
	s.titleLabel = widget.NewLabelWithStyle(i18n.Text("orders_title"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	topBar := container.NewPadded(
		container.NewVBox(
			container.NewHBox(
				s.titleLabel,
				layout.NewSpacer(),
				langControls,
				layout.NewSpacer(),
				s.logoutButton,
			),
			refreshControls,
		),
	)
	
	orderListContainer := container.NewBorder(
		nil, nil, nil, nil,
		container.NewPadded(s.createOrdersList()),
	)
	
	splitContainer := container.NewHSplit(
		orderListContainer,
		s.detailsContainer,
	)
	splitContainer.SetOffset(0.3) 
	
	content := container.NewBorder(
		container.NewVBox(
			topBar,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		container.NewPadded(splitContainer),
	)
	
	return content
}


func (s *OrdersScreen) PerformInitialSetup() {
	// Call fetchOrders to populate the list when the screen is first shown
	s.fetchOrders()
	// Set up other initial configurations like key listeners
	s.window.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyEscape && s.refreshTimer != nil {
			s.refreshTimer.Stop()
		}
	})
	s.updateLanguageUI() // Ensure UI is updated with current language
}


func (s *OrdersScreen) playNotificationSound() {
	if err := utils.PlaySound("assets/horn.mp3"); err != nil {
		
	}
}


func (s *OrdersScreen) handleRefreshIntervalChange(selected string) {
	
	if s.refreshTimer != nil {
		s.refreshTimer.Stop()
		s.refreshTimer = nil
	}
	
	
	switch selected {
	case i18n.Text("off"):
		s.isAutoRefresh = false
		return
	case "5 " + i18n.Text("minutes"):
		s.refreshInterval = time.Minute * 5
	case "10 " + i18n.Text("minutes"):
		s.refreshInterval = time.Minute * 10
	case "15 " + i18n.Text("minutes"):
		s.refreshInterval = time.Minute * 15
	case "30 " + i18n.Text("minutes"):
		s.refreshInterval = time.Minute * 30
	case "60 " + i18n.Text("minutes"):
		s.refreshInterval = time.Minute * 60
	}
	
	s.isAutoRefresh = true
	
	
	s.startRefreshTimer()
	
	
	fyne.CurrentApp().SendNotification(&fyne.Notification{
		Title:   i18n.Text("auto_refresh"),
		Content: fmt.Sprintf(i18n.Text("auto_refresh_set"), selected),
	})
}


func (s *OrdersScreen) startRefreshTimer() {
	if s.refreshTimer != nil {
		s.refreshTimer.Stop()
	}
	
	s.refreshTimer = time.AfterFunc(s.refreshInterval, func() {
		
		fyne.Do(func() {
			s.fetchOrders()
			
			
			s.startRefreshTimer()
		})
	})
}


func (s *OrdersScreen) createOrdersList() *widget.List {
	s.ordersList = widget.NewList(
		func() int {
			return len(s.orders)
		},
		func() fyne.CanvasObject {
			return container.NewPadded(
				container.NewVBox(
					widget.NewLabelWithStyle("Model", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
					widget.NewLabel("Reference No"),
					widget.NewLabel("Status"),
					widget.NewSeparator(),
				),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(s.orders) {
				return
			}
			order := s.orders[id]
			container := item.(*fyne.Container).Objects[0].(*fyne.Container)
			
			
			modelLabel := container.Objects[0].(*widget.Label)
			modelLabel.SetText(fmt.Sprintf("Model: %s", order.Order.ModelCode))
			
			refLabel := container.Objects[1].(*widget.Label)
			refLabel.SetText(fmt.Sprintf("Ref: %s", order.Order.ReferenceNumber))
			
			statusLabel := container.Objects[2].(*widget.Label)
			statusLabel.SetText(fmt.Sprintf("Status: %s", order.Order.OrderStatus))
		},
	)
	
	
	s.ordersList.OnSelected = func(id widget.ListItemID) {
		if id < len(s.orders) {
			fyne.Do(func() {
				s.showOrderDetails(s.orders[id])
			})
		}
	}
	
	return s.ordersList
}


func (s *OrdersScreen) processDifferences(differences []string, newOrders []tesla.DetailedOrder) {
	
	s.changedFields = make(map[string]bool)
	
	
	for _, diff := range differences {
		
		if orderRef, field, found := extractOrderFieldFromDiff(diff, s.orders); found {
			
			key := fmt.Sprintf("%s_%s", field, orderRef)
			s.changedFields[key] = true
		}
	}
	
	
	if len(s.changedFields) == 0 && len(newOrders) > 0 {
		
		orderRef := newOrders[0].Order.ReferenceNumber
		s.changedFields["Status_"+orderRef] = true
		s.changedFields["VIN_"+orderRef] = true
		s.changedFields["DeliveryWindow_"+orderRef] = true
	}
}


func extractOrderFieldFromDiff(diff string, orders []tesla.DetailedOrder) (string, string, bool) {
	var orderRef string
	
	
	for i := 0; i < len(diff)-6; i++ {
		if i+6 <= len(diff) && diff[i:i+2] == "RN" {
			potential := diff[i : i+8] 
			if isReferenceNumber(potential) {
				orderRef = potential
				break
			}
		}
	}
	
	
	if orderRef == "" && len(orders) > 0 {
		orderRef = orders[0].Order.ReferenceNumber
	}
	
	
	if containsAny(diff, []string{"modelCode", "model"}) {
		return orderRef, "Model", true
	}
	
	
	if containsAny(diff, []string{"orderStatus", "status", "Status"}) {
		return orderRef, "Status", true
	}
	
	
	if containsAny(diff, []string{"vin", "VIN"}) {
		return orderRef, "VIN", true
	}
	
	
	if containsAny(diff, []string{"vehicleOdometer", "odometer", "Odometer"}) {
		return orderRef, "Odometer", true
	}
	
	
	if containsAny(diff, []string{"reservationDate", "reservation"}) {
		return orderRef, "ReservationDate", true
	}
	
	
	if containsAny(diff, []string{"orderBookedDate", "orderDate"}) {
		return orderRef, "OrderBookedDate", true
	}
	
	
	if containsAny(diff, []string{"vehicleRoutingLocation", "routingLocation", "location"}) {
		return orderRef, "RoutingLocation", true
	}
	
	
	if containsAny(diff, []string{"deliveryWindow", "delivery", "window"}) {
		return orderRef, "DeliveryWindow", true
	}
	
	
	if containsAny(diff, []string{"etaToDeliveryCenter", "eta", "ETA"}) {
		return orderRef, "ETAToDeliveryCenter", true
	}
	
	
	if containsAny(diff, []string{"apptDateTime", "appointment", "Appointment"}) {
		return orderRef, "DeliveryAppointment", true
	}
	
	return "", "", false
}


func isReferenceNumber(str string) bool {
	if len(str) >= 8 && str[0:2] == "RN" {
		return true
	}
	return false
}


func containsAny(text string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(strings.ToLower(text), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}


func (s *OrdersScreen) createHighlightedLabel(text string, key string) fyne.CanvasObject {
	isChanged := s.changedFields[key]
	
	if isChanged {
		
		highlightColor := &color.RGBA{R: 255, G: 200, B: 0, A: 100} 
		bg := canvas.NewRectangle(highlightColor)
		bg.SetMinSize(fyne.NewSize(200, 30))
		
		label := widget.NewLabel(text)
		label.TextStyle = fyne.TextStyle{Bold: true}
		
		return container.NewStack(bg, container.NewPadded(label))
	}
	
	
	return widget.NewLabel(text)
}


func (s *OrdersScreen) showOrderDetails(order tesla.DetailedOrder) {
	
	s.currentOrderDetail = order
	
	info := s.orderManager.ExtractOrderInfo(order)
	
	
	titleStyle := fyne.TextStyle{Bold: true}
	sectionTitleSize := float32(1.1) 
	
	
	s.mainDetailTitle = widget.NewLabelWithStyle(i18n.Text("order_details"), fyne.TextAlignCenter, titleStyle)
	s.mainDetailTitle.TextStyle.Bold = true
	
	
	orderContainer := container.NewVBox()
	orderForm := widget.NewForm()
	
	
	orderTitle := canvas.NewText(i18n.Text("order_information"), theme.ForegroundColor())
	orderTitle.TextStyle = titleStyle
	orderTitle.TextSize = theme.TextSize() * sectionTitleSize
	
	
	s.orderTitles = make([]*canvas.Text, 0)
	s.orderTitles = append(s.orderTitles, orderTitle)
	
	
	orderForm.Append(i18n.Text("model"), s.createHighlightedLabel(info["Model"], "Model_"+order.Order.ReferenceNumber))
	
	
	orderForm.Append(i18n.Text("order_number"), widget.NewLabel(info["OrderID"]))
	
	
	orderForm.Append(i18n.Text("status"), s.createHighlightedLabel(info["Status"], "Status_"+order.Order.ReferenceNumber))
	
	
	orderForm.Append(i18n.Text("vin"), s.createHighlightedLabel(info["VIN"], "VIN_"+order.Order.ReferenceNumber))
	
	
	kmText := fmt.Sprintf("%s %s", info["VehicleOdometer"], info["VehicleOdometerType"])
	orderForm.Append(i18n.Text("vehicle_odometer"), s.createHighlightedLabel(kmText, "Odometer_"+order.Order.ReferenceNumber))
	
	
	orderContainer.Add(orderTitle)
	orderContainer.Add(widget.NewSeparator())
	orderContainer.Add(container.NewPadded(orderForm))
	
	
	reservationContainer := container.NewVBox()
	reservationForm := widget.NewForm()
	
	
	reservationTitle := canvas.NewText(i18n.Text("reservation_information"), theme.ForegroundColor())
	reservationTitle.TextStyle = titleStyle
	reservationTitle.TextSize = theme.TextSize() * sectionTitleSize
	s.orderTitles = append(s.orderTitles, reservationTitle)
	
	
	reservationForm.Append(i18n.Text("reservation_date"), 
		s.createHighlightedLabel(info["ReservationDate"], "ReservationDate_"+order.Order.ReferenceNumber))
	
	
	reservationForm.Append(i18n.Text("order_date"), 
		s.createHighlightedLabel(info["OrderBookedDate"], "OrderBookedDate_"+order.Order.ReferenceNumber))
	
	
	reservationContainer.Add(reservationTitle)
	reservationContainer.Add(widget.NewSeparator())
	reservationContainer.Add(container.NewPadded(reservationForm))
	
	
	deliveryContainer := container.NewVBox()
	deliveryForm := widget.NewForm()
	
	
	deliveryTitle := canvas.NewText(i18n.Text("delivery_information"), theme.ForegroundColor())
	deliveryTitle.TextStyle = titleStyle
	deliveryTitle.TextSize = theme.TextSize() * sectionTitleSize
	s.orderTitles = append(s.orderTitles, deliveryTitle)
	
	routingLocationID, _ := strconv.Atoi(info["VehicleRoutingLocation"])
	teslaStore := tesla.GetTeslaStoreByID(routingLocationID)
	
	locationText := fmt.Sprintf("%s (%s)", info["VehicleRoutingLocation"], teslaStore.Label)
	
	
	deliveryForm.Append(i18n.Text("delivery_location"), 
		s.createHighlightedLabel(locationText, "RoutingLocation_"+order.Order.ReferenceNumber))
	
	
	deliveryForm.Append(i18n.Text("delivery_window"), 
		s.createHighlightedLabel(info["DeliveryWindow"], "DeliveryWindow_"+order.Order.ReferenceNumber))
	
	
	deliveryForm.Append(i18n.Text("estimated_arrival"), 
		s.createHighlightedLabel(info["ETAToDeliveryCenter"], "ETAToDeliveryCenter_"+order.Order.ReferenceNumber))
	
	
	deliveryForm.Append(i18n.Text("delivery_appointment"), 
		s.createHighlightedLabel(info["DeliveryAppointment"], "DeliveryAppointment_"+order.Order.ReferenceNumber))
	
	
	deliveryContainer.Add(deliveryTitle)
	deliveryContainer.Add(widget.NewSeparator())
	deliveryContainer.Add(container.NewPadded(deliveryForm))
	
	
	paymentContainer := container.NewVBox()
	paymentForm := widget.NewForm()
	
	
	paymentTitle := canvas.NewText(i18n.Text("payment_details"), theme.ForegroundColor())
	paymentTitle.TextStyle = titleStyle
	paymentTitle.TextSize = theme.TextSize() * sectionTitleSize
	s.orderTitles = append(s.orderTitles, paymentTitle)
	
	
	
	var reservationAmount float64 = 0
	
	
	if registration, ok := order.Details.Tasks["registration"].(map[string]interface{}); ok {
		if orderDetails, ok := registration["orderDetails"].(map[string]interface{}); ok {
			if amount, ok := orderDetails["reservationAmountReceived"].(float64); ok {
				reservationAmount = amount
				paymentForm.Append(i18n.Text("reservation_amount"), widget.NewLabel(formatCurrency(amount, "TRY")))
			}
		}
	}
	
	if finalPayment, ok := order.Details.Tasks["finalPayment"].(map[string]interface{}); ok {
		
		var totalAmount float64 = reservationAmount
		var amountDue float64
		
		if finalPayment["amountDue"] != nil {
			if amount, ok := finalPayment["amountDue"].(float64); ok {
				amountDue = amount
				totalAmount += amount
			}
		}
		
		
		paymentForm.Append(i18n.Text("total_price"), widget.NewLabel(formatCurrency(totalAmount, "TRY")))
		
		
		paymentForm.Append(i18n.Text("remaining_amount"), widget.NewLabel(formatCurrency(amountDue, "TRY")))
		
		
		if status, ok := finalPayment["status"].(string); ok {
			paymentForm.Append(i18n.Text("payment_status"), widget.NewLabel(formatPaymentStatus(status)))
		}
	}
	
	
	paymentContainer.Add(paymentTitle)
	paymentContainer.Add(widget.NewSeparator())
	paymentContainer.Add(container.NewPadded(paymentForm))
	
	verticalSpacer := func() fyne.CanvasObject {
		rect := canvas.NewRectangle(color.Transparent)
		rect.SetMinSize(fyne.NewSize(0, theme.Padding())) 
		return rect
	}
	
	
	content := container.NewVBox(
		s.mainDetailTitle,
		widget.NewSeparator(), 
		orderContainer,
		verticalSpacer(),
		reservationContainer,
		verticalSpacer(),
		deliveryContainer,
		verticalSpacer(),
		paymentContainer,
	)
	
	
	scrollContent := container.NewScroll(content)
	
	
	paddedScrollContent := container.NewPadded(scrollContent)
	
	
	fyne.Do(func() {
		s.detailsContainer.Objects = []fyne.CanvasObject{paddedScrollContent} 
		s.detailsContainer.Refresh()
	})
}


func formatCurrency(amount float64, currency string) string {
	return fmt.Sprintf("%.2f %s", amount, currency)
}


func formatPaymentStatus(status string) string {
	switch status {
	case "MAKE_YOUR_FINAL_PAYMENT":
		return i18n.Text("waiting_for_final_payment")
	case "PAYMENT_RECEIVED":
		return i18n.Text("payment_received")
	case "PAYMENT_PROCESSING":
		return i18n.Text("payment_processing")
	default:
		return status
	}
}


func (s *OrdersScreen) fetchOrders() {
	
	progress := dialog.NewProgressInfinite(i18n.Text("loading_orders"), i18n.Text("fetching_orders"), s.window)
	progress.Show()
	
	
	go func() {
		
		oldOrders, _ := s.orderManager.LoadOrdersFromFile()
		
		
		newOrders, err := s.orderManager.GetDetailedOrders()
		if err != nil {
			fyne.Do(func() {
				progress.Hide()
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   i18n.Text("error"),
					Content: fmt.Sprintf(i18n.Text("error_fetching_orders"), err),
				})
			})
			return
		}
		
		
		fyne.Do(func() {
			
			s.orders = newOrders
		})
		
		
		hasChanges := false
		var differences []string
		
		if oldOrders != nil && len(oldOrders) > 0 {
			differences = s.orderManager.CompareOrders(oldOrders, newOrders)
			if len(differences) > 0 {
				hasChanges = true
				
				
				s.processDifferences(differences, newOrders)
				
				
				s.playNotificationSound()
				
				
				fyne.Do(func() {
					
					fyne.CurrentApp().SendNotification(&fyne.Notification{
						Title:   i18n.Text("changes"),
						Content: i18n.Text("order_changes_detected"),
					})
				})
			}
		}
		
		
		s.orderManager.SaveOrdersToFile(newOrders)
		
		
		s.lastRefreshTime = time.Now()
		
		
		fyne.Do(func() {
			
			statusText := fmt.Sprintf("%s: %s", i18n.Text("last_refresh"), s.lastRefreshTime.Format("15:04:05"))
			if hasChanges {
				statusText += " (" + i18n.Text("changes_detected") + ")"
			}
			s.refreshStatusLabel.SetText(statusText)
			
			
			progress.Hide()
			s.ordersList.Refresh()
			
			
			if len(s.orders) > 0 {
				s.ordersList.Select(0)
				s.showOrderDetails(s.orders[0])
			}
		})
	}()
}


func (s *OrdersScreen) createLanguageSelector() *widget.Select {
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


func (s *OrdersScreen) updateLanguageUI() {
	
	s.window.SetTitle(i18n.Text("app_title"))
	
	
	if s.titleLabel != nil {
		s.titleLabel.SetText(i18n.Text("orders_title"))
	}
	
	
	if s.langSelect != nil {
		
		s.langSelect.Options = []string{i18n.Text("english"), i18n.Text("turkish")}
		
		
		if i18n.CurrentLang == i18n.LangEnglish {
			s.langSelect.SetSelected(i18n.Text("english"))
		} else {
			s.langSelect.SetSelected(i18n.Text("turkish"))
		}
	}
	
	
	if s.logoutButton != nil {
		s.logoutButton.SetText(i18n.Text("logout"))
	}
	
	if s.refreshButton != nil {
		s.refreshButton.SetText(i18n.Text("refresh"))
	}
	
	
	if s.autoRefreshLabel != nil {
		s.autoRefreshLabel.SetText(i18n.Text("auto_refresh") + ":")
	}
	
	if s.refreshStatusLabel != nil {
		
		refreshTimeText := s.lastRefreshTime.Format("15:04:05")
		s.refreshStatusLabel.SetText(i18n.Text("last_refresh") + ": " + refreshTimeText)
	}
	
	
	if s.noOrdersLabel != nil {
		s.noOrdersLabel.SetText(i18n.Text("select_order"))
	}
	
	
	if s.refreshSelect != nil {
		
		newOptions := []string{
			i18n.Text("off"), 
			"5 " + i18n.Text("minutes"), 
			"10 " + i18n.Text("minutes"), 
			"15 " + i18n.Text("minutes"), 
			"30 " + i18n.Text("minutes"), 
			"60 " + i18n.Text("minutes"),
		}
		
		
		s.refreshSelect.Options = newOptions
		
		
		if len(newOptions) > 0 {
			s.refreshSelect.SetSelected(newOptions[0])
		}
	}
	
	
	if s.mainDetailTitle != nil {
		s.mainDetailTitle.SetText(i18n.Text("order_details"))
		
		
		if s.orderTitles != nil && len(s.orderTitles) > 0 {
			if len(s.orderTitles) > 0 && s.orderTitles[0] != nil {
				s.orderTitles[0].Text = i18n.Text("order_information")
			}
			if len(s.orderTitles) > 1 && s.orderTitles[1] != nil {
				s.orderTitles[1].Text = i18n.Text("reservation_information")
			}
			if len(s.orderTitles) > 2 && s.orderTitles[2] != nil {
				s.orderTitles[2].Text = i18n.Text("delivery_information")
			}
			if len(s.orderTitles) > 3 && s.orderTitles[3] != nil {
				s.orderTitles[3].Text = i18n.Text("payment_details")
			}
		}
		
		
		if !reflect.DeepEqual(s.currentOrderDetail, tesla.DetailedOrder{}) {
			s.showOrderDetails(s.currentOrderDetail)
		}
	}
	
	
	if s.ordersList != nil {
		s.ordersList.Refresh()
	}
	// Wrap UI refresh calls in fyne.Do to ensure they run on the main Fyne thread.
	fyne.Do(func() {
		if s.titleLabel != nil {
			s.titleLabel.Refresh()
		}
		if s.langSelect != nil {
			s.langSelect.Refresh()
		}
		if s.logoutButton != nil {
			s.logoutButton.Refresh()
		}
		if s.refreshButton != nil {
			s.refreshButton.Refresh()
		}
		if s.autoRefreshLabel != nil {
			s.autoRefreshLabel.Refresh()
		}
		if s.refreshStatusLabel != nil {
			s.refreshStatusLabel.Refresh()
		}
		if s.noOrdersLabel != nil {
			s.noOrdersLabel.Refresh()
		}
		if s.refreshSelect != nil {
			s.refreshSelect.Refresh()
		}
		if s.mainDetailTitle != nil {
			s.mainDetailTitle.Refresh()
			if s.orderTitles != nil {
				for _, title := range s.orderTitles {
					if title != nil {
						title.Refresh()
					}
				}
			}
		}
		if s.ordersList != nil {
			s.ordersList.Refresh()
		}
		if !reflect.DeepEqual(s.currentOrderDetail, tesla.DetailedOrder{}) && s.detailsContainer != nil {
			// showOrderDetails itself has fyne.Do, so we might not need to refresh the whole container here
			// but if individual components were not refreshed, this might be needed.
			// For now, assume showOrderDetails handles its own refresh.
		}
	})
} 