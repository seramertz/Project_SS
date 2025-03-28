package distributor

import (
	"Driver-go/config"
	"Driver-go/elevio"
	"strconv"
	"time"
)

//---Utility functions used by distributor for elevator updates and communication---

func broadcastElevatorState(elevators []*config.ElevatorDistributor, ch_transmit chan<- []config.ElevatorDistributor) {
	temporaryElevators := make([]config.ElevatorDistributor, 0)
	for _, elevator := range elevators {
		temporaryElevators = append(temporaryElevators, *elevator)
	}
	ch_transmit <- temporaryElevators
	time.Sleep(25 * time.Millisecond)
}

func reinitializeElevator(elevators []*config.ElevatorDistributor, id int) {
	for _, elev := range elevators {
		if elev.ID == strconv.Itoa(id) {
			*elev = elevatorDistributorInit(strconv.Itoa(id))
			break
		}
	}
}

// updateElevators updates the local elevator state based on received network data.
// It ensures that:
// - Orders are updated only if they were not already confirmed.
// - Floor position, direction, and behavior are synchronized.
// - If the elevator is available, it assigns new network orders.
func updateElevators(elevators []*config.ElevatorDistributor, newElevators []config.ElevatorDistributor) {
	if elevators[config.LocalElevator].ID != newElevators[config.LocalElevator].ID {

		// Find the elevator corresponding to the ID in the local list and update its state
		for _, elev := range elevators {
			if elev.ID == newElevators[config.LocalElevator].ID {
				for floor := range elev.Requests {
					for button := range elev.Requests[floor] {
						// Update requests only if they are not already confirmed
						if !(elev.Requests[floor][button] == config.Confirmed) &&
							(newElevators[config.LocalElevator].Requests[floor][button] == config.Order) {

							elev.Requests[floor][button] = newElevators[config.LocalElevator].Requests[floor][button]
						}

						// Synchronize elevator state with the network update
						elev.Floor = newElevators[config.LocalElevator].Floor
						elev.Direction = newElevators[config.LocalElevator].Direction
						elev.Behaviour = newElevators[config.LocalElevator].Behaviour
					}
				}
			}
		}
		// Ensure the local elevator updates its orders if it's available
		for _, newElev := range newElevators {
			if newElev.ID == elevators[config.LocalElevator].ID {
				for floor := range newElev.Requests {
					for button := range newElev.Requests[floor] {
						// Assign new orders only if the elevator is available
						if (elevators[config.LocalElevator].Behaviour != config.Unavailable) &&
							(newElev.Requests[floor][button] == config.Order) {

							(*elevators[config.LocalElevator]).Requests[floor][button] = config.Order
						}
					}
				}
			}
		}
	}
}

func addNewElevator(elevators *[]*config.ElevatorDistributor, newElevator config.ElevatorDistributor) {
	tempElev := new(config.ElevatorDistributor)
	*tempElev = elevatorDistributorInit(newElevator.ID)
	(*tempElev).Behaviour = newElevator.Behaviour
	(*tempElev).Direction = newElevator.Direction
	(*tempElev).Floor = newElevator.Floor

	for floor := range tempElev.Requests {
		for button := range tempElev.Requests[floor] {
			tempElev.Requests[floor][button] = newElevator.Requests[floor][button]
		}
	}
	*elevators = append(*elevators, tempElev)
}

func setElevatorLights(elevators []*config.ElevatorDistributor, elevatorID int) {
	// Set hall lights
	for button := 0; button < config.NumButtons-1; button++ {
		for floor := 0; floor < config.NumFloors; floor++ {
			isLight := false
			for _, elev := range elevators {
				if elev.Requests[floor][button] == config.Confirmed {
					isLight = true
				}
			}
			elevio.SetButtonLamp(elevio.ButtonType(button), floor, isLight)
		}
	}

	// Set cab lights
	for floor := 0; floor < config.NumFloors; floor++ {
		for _, elev := range elevators {
			if elev.ID == strconv.Itoa(elevatorID) && elev.Requests[floor][elevio.BT_Cab] == config.Confirmed {
				elevio.SetButtonLamp(elevio.BT_Cab, floor, true)
			}
		}
	}
}
