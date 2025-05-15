package tesla

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

type Order struct {
	ReferenceNumber string `json:"referenceNumber"`
	OrderStatus     string `json:"orderStatus"`
	ModelCode       string `json:"modelCode"`
	VIN             string `json:"vin,omitempty"`
}

type OrderDetails struct {
	Tasks map[string]interface{} `json:"tasks"`
}

type DetailedOrder struct {
	Order   Order        `json:"order"`
	Details OrderDetails `json:"details"`
}

type OrderManager struct {
	Auth *TeslaAuth
}

func NewOrderManager(auth *TeslaAuth) *OrderManager {
	return &OrderManager{
		Auth: auth,
	}
}

func (m *OrderManager) RetrieveOrders() ([]Order, error) {
	resp, err := m.Auth.Client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", m.Auth.AccessToken)).
		Get("https://owner-api.teslamotors.com/api/1/users/orders")
	
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to retrieve orders: %s", resp.String())
	}
	
	var response struct {
		Response []Order `json:"response"`
	}
	
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return nil, err
	}
	
	return response.Response, nil
}

func (m *OrderManager) GetOrderDetails(orderID string) (*OrderDetails, error) {
	resp, err := m.Auth.Client.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", m.Auth.AccessToken)).
		Get(fmt.Sprintf("https://akamai-apigateway-vfx.tesla.com/tasks?deviceLanguage=en&deviceCountry=DE&referenceNumber=%s&appVersion=%s", 
			orderID, AppVersion))
	
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get order details: %s", resp.String())
	}
	
	var details OrderDetails
	if err := json.Unmarshal(resp.Body(), &details); err != nil {
		return nil, err
	}
	
	return &details, nil
}

func (m *OrderManager) GetDetailedOrders() ([]DetailedOrder, error) {
	orders, err := m.RetrieveOrders()
	if err != nil {
		return nil, err
	}
	
	detailedOrders := make([]DetailedOrder, 0, len(orders))
	
	for _, order := range orders {
		details, err := m.GetOrderDetails(order.ReferenceNumber)
		if err != nil {
			return nil, err
		}
		
		detailedOrder := DetailedOrder{
			Order:   order,
			Details: *details,
		}
		
		detailedOrders = append(detailedOrders, detailedOrder)
	}
	
	return detailedOrders, nil
}

func (m *OrderManager) SaveOrdersToFile(orders []DetailedOrder) error {
	data, err := json.Marshal(orders)
	if err != nil {
		return err
	}
	
	return os.WriteFile(OrdersFile, data, 0600)
}

func (m *OrderManager) LoadOrdersFromFile() ([]DetailedOrder, error) {
	if _, err := os.Stat(OrdersFile); os.IsNotExist(err) {
		return nil, nil
	}
	
	data, err := os.ReadFile(OrdersFile)
	if err != nil {
		return nil, err
	}
	
	var orders []DetailedOrder
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}
	
	return orders, nil
}

func (m *OrderManager) CompareOrders(old, new []DetailedOrder) []string {
	differences := []string{}
	
	for i, oldOrder := range old {
		if i < len(new) {
			diff := compareMaps(extractMap(oldOrder), extractMap(new[i]), "")
			differences = append(differences, diff...)
		} else {
			differences = append(differences, fmt.Sprintf("Removed order %s", oldOrder.Order.ReferenceNumber))
		}
	}
	
	for i := len(old); i < len(new); i++ {
		differences = append(differences, fmt.Sprintf("Added order %s", new[i].Order.ReferenceNumber))
	}
	
	return differences
}

func extractMap(order DetailedOrder) map[string]interface{} {
	data, _ := json.Marshal(order)
	result := map[string]interface{}{}
	json.Unmarshal(data, &result)
	return result
}

func compareMaps(old, new map[string]interface{}, path string) []string {
	differences := []string{}
	
	for key, oldValue := range old {
		if newValue, exists := new[key]; !exists {
			differences = append(differences, fmt.Sprintf("Removed key '%s%s'", path, key))
		} else {
			oldMap, oldIsMap := oldValue.(map[string]interface{})
			newMap, newIsMap := newValue.(map[string]interface{})
			
			if oldIsMap && newIsMap {
				diff := compareMaps(oldMap, newMap, fmt.Sprintf("%s%s.", path, key))
				differences = append(differences, diff...)
			} else if !areValuesEqual(oldValue, newValue) {
				differences = append(differences, fmt.Sprintf("Changed value at '%s%s': %v -> %v", path, key, formatValue(oldValue), formatValue(newValue)))
			}
		}
	}
	
	for key, newValue := range new {
		if _, exists := old[key]; !exists {
			differences = append(differences, fmt.Sprintf("Added key '%s%s': %v", path, key, formatValue(newValue)))
		}
	}
	
	return differences
}

func areValuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	aType := reflect.TypeOf(a)
	bType := reflect.TypeOf(b)

	if aType != bType {
		return false
	}

	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		if len(aVal) == 0 && len(bVal) == 0 {
			return true
		}
		return len(aVal) == len(bVal)

	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(aVal) == 0 && len(bVal) == 0 {
			return true
		}
		return len(aVal) == len(bVal)

	default:
		return reflect.DeepEqual(a, b)
	}
}

func formatValue(v interface{}) string {
	if v == nil {
		return "nil"
	}
	
	if s, ok := v.(string); ok {
		if len(s) > 30 {
			return fmt.Sprintf("%s...", s[:30])
		}
		return s
	}
	
	return fmt.Sprintf("%v", v)
}

func (m *OrderManager) ExtractOrderInfo(order DetailedOrder) map[string]string {
	info := make(map[string]string)
	
	info["OrderID"] = order.Order.ReferenceNumber
	info["Status"] = order.Order.OrderStatus
	info["Model"] = order.Order.ModelCode
	info["VIN"] = order.Order.VIN
	
	tasks := order.Details.Tasks
	
	if registration, ok := tasks["registration"].(map[string]interface{}); ok {
		if orderDetails, ok := registration["orderDetails"].(map[string]interface{}); ok {
			setIfString(orderDetails, "reservationDate", &info, "ReservationDate")
			setIfString(orderDetails, "orderBookedDate", &info, "OrderBookedDate")
			setIfString(orderDetails, "vehicleOdometer", &info, "VehicleOdometer")
			setIfString(orderDetails, "vehicleOdometerType", &info, "VehicleOdometerType")
			setIfString(orderDetails, "vehicleRoutingLocation", &info, "VehicleRoutingLocation")
		}
	}
	
	if scheduling, ok := tasks["scheduling"].(map[string]interface{}); ok {
		setIfString(scheduling, "deliveryWindowDisplay", &info, "DeliveryWindow")
		setIfString(scheduling, "apptDateTimeAddressStr", &info, "DeliveryAppointment")
	}
	
	if finalPayment, ok := tasks["finalPayment"].(map[string]interface{}); ok {
		if data, ok := finalPayment["data"].(map[string]interface{}); ok {
			setIfString(data, "etaToDeliveryCenter", &info, "ETAToDeliveryCenter")
		}
	}
	
	return info
}

func setIfString(data map[string]interface{}, key string, info *map[string]string, infoKey string) {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			(*info)[infoKey] = s
		} else if f, ok := v.(float64); ok {
			if key == "vehicleOdometer" {
				(*info)[infoKey] = fmt.Sprintf("%v", f)
			} else {
				(*info)[infoKey] = fmt.Sprintf("%.2f", f)
			}
		}
	} else {
		(*info)[infoKey] = "N/A"
	}
} 