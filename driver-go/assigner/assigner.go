package assigner

import (
	"Driver-go/config"
	"Driver-go/assigner/cost"
	"Driver-go/elevio"
	"strconv"
)

// Checks for unavailable elevators and reassigns eventual orders if an elevator is unavailable.
func ReassignOrders(elevators []*config.ElevatorDistributor, ch_newLocalOrder chan elevio.ButtonEvent) {
	lowestID := findLowestID(elevators)
	for _, elev := range elevators {
		if elev.Behaviour == config.Unavailable {
			for floor := range elev.Requests {
				for button := 0; button < len(elev.Requests[floor])-1; button++ {
					if elev.Requests[floor][button] == config.Order ||
						elev.Requests[floor][button] == config.Confirmed {
						if elevators[config.LocalElevator].ID == strconv.Itoa(lowestID) {
							ch_newLocalOrder <- elevio.ButtonEvent{
								Floor:  floor,
								Button: elevio.ButtonType(button)}
						}
					}
				}
			}
		}
	}
}
// Finds and returns the lowest ID of the elevators
func findLowestID(elevators []*config.ElevatorDistributor) int {
	lowID := config.MaxCost
	for _, elev := range elevators {
		if elev.Behaviour != config.Unavailable {
			ID, _ := strconv.Atoi(elev.ID)
			if ID < lowID {
				lowID = ID
			}
		}
	}
	return lowID
}

// Assignes new order to the right elevator depending on a cost function.
func AssignOrder(elevators []*config.ElevatorDistributor, order elevio.ButtonEvent){
	if len(elevators) < 2 || order.Button == elevio.BT_Cab {
		elevators[config.LocalElevator].Requests[order.Floor][order.Button] = config.Order
		return
	}
	minCost := config.MaxAssignment
	elevCost := 0
	var minElev *config.ElevatorDistributor
	for _, elev := range elevators {
		elevCost = cost.Cost(elev, order)
		if elevCost < minCost {
			minCost = elevCost
			minElev = elev
		}
	}
	(*minElev).Requests[order.Floor][order.Button] = config.Order
}

// Remove completed orders from the elevator
func RemoveCompletedOrders(elevators []*config.ElevatorDistributor) {
	for _, elev := range elevators {
		for floor := range elev.Requests {
			for button := range elev.Requests[floor] {
				if elev.Requests[floor][button] == config.Complete {
					elev.Requests[floor][button] = config.None
				}
			}
		}
	}
}

// Extracts a new order from the elevator
func ConfirmedNewOrder(elev *config.ElevatorDistributor) *config.Requests {
	for floor := range elev.Requests {
		for button := 0; button < len(elev.Requests[floor]); button++ {
			if elev.Requests[floor][button] == config.Order {
				elev.Requests[floor][button] = config.Confirmed
				tempOrder := new(config.Requests)
				*tempOrder = config.Requests{
					Floor:  floor,
					Button: config.ButtonType(button)}
				return tempOrder
			}
		}
	}
	return nil
}