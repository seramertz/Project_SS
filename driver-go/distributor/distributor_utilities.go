package distributor

import (
	"Driver-go/config"
	"Driver-go/elevio"
	"strconv"
	"time"
)

//Utility functions used by the distributor for elevator updates and communication

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

func updateElevators(elevators []*config.ElevatorDistributor, newElevators []config.ElevatorDistributor) {
	if elevators[config.LocalElevator].ID != newElevators[config.LocalElevator].ID {
		for _, elev := range elevators {
			if elev.ID == newElevators[config.LocalElevator].ID {
				for floor := range elev.Requests {
					for button := range elev.Requests[floor] {
						if (!(elev.Requests[floor][button] == config.Confirmed) &&
						(newElevators[config.LocalElevator].Requests[floor][button] == config.Order)){

							elev.Requests[floor][button] = newElevators[config.LocalElevator].Requests[floor][button]
						}
						elev.Floor = newElevators[config.LocalElevator].Floor
						elev.Direction = newElevators[config.LocalElevator].Direction
						elev.Behaviour = newElevators[config.LocalElevator].Behaviour
					}
				}
			}
		}
		for _, newElev := range newElevators {
			if newElev.ID == elevators[config.LocalElevator].ID {
				for floor := range newElev.Requests {
					for button := range newElev.Requests[floor] {
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
	for floor := 0; floor < config.NumFloors; floor++ {
		for _, elev := range elevators {
			if elev.ID == strconv.Itoa(elevatorID) && elev.Requests[floor][elevio.BT_Cab] == config.Confirmed {
				elevio.SetButtonLamp(elevio.BT_Cab, floor, true)
			}
		}
	}
}