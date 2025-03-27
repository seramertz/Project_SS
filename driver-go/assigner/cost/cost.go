package cost

import (
	"Driver-go/config"
	"Driver-go/elevio"
)

// Cost function that calculates the cost of assigning an order to an elevator
// Based on the distributed hall assigner from github repo
func Cost(elev *config.ElevatorDistributor, req elevio.ButtonEvent) int {
	highestDuration := config.MaxCost
	if elev.Behaviour != config.Unavailable {
		e := new(config.ElevatorDistributor)
		*e = *elev
		e.Requests[req.Floor][req.Button] = config.Confirmed

		duration := 0

		switch e.Behaviour {
		case config.Idle:
			distributorRequestChooseDirection(e)
			if e.Direction == config.Stop {
				return duration
			}
		case config.Moving:
			duration += config.TravelTime / 2
			e.Floor += int(e.Direction)
		case config.DoorOpen:
			duration -= config.DoorOpenDuration / 2
		}

		for {
			if distributorRequestShouldStop(*e) {
				distributorRequestClearAtCurrentFloor(e)
				duration += config.DoorOpenDuration
				distributorRequestChooseDirection(e)
				if e.Direction == config.Stop {
					return duration
				}
			}
			e.Floor += int(e.Direction)
			duration += config.TravelTime
		}
	}
	return highestDuration
}

// ---Request functions for elevator distributor object---

func distributorRequestsAbove(elev config.ElevatorDistributor) bool {
	for f := elev.Floor + 1; f < config.NumFloors; f++ {
		for btn := range elev.Requests[f] {
			if elev.Requests[f][btn] == config.Confirmed {
				return true
			}
		}
	}
	return false
}

func distributorRequestsBelow(elev config.ElevatorDistributor) bool {
	for f := 0; f < elev.Floor; f++ {
		for btn := range elev.Requests[f] {
			if elev.Requests[f][btn] == config.Confirmed {
				return true
			}
		}
	}
	return false
}

func distributorRequestClearAtCurrentFloor(elev *config.ElevatorDistributor) {
	elev.Requests[elev.Floor][int(elevio.BT_Cab)] = config.None
	switch {
	case elev.Direction == config.Up:
		elev.Requests[elev.Floor][int(elevio.BT_HallUp)] = config.None
		if !distributorRequestsAbove(*elev) {
			elev.Requests[elev.Floor][int(elevio.BT_HallDown)] = config.None
		}
	case elev.Direction == config.Down:
		elev.Requests[elev.Floor][int(elevio.BT_HallDown)] = config.None
		if !distributorRequestsBelow(*elev) {
			elev.Requests[elev.Floor][int(elevio.BT_HallUp)] = config.None
		}
	}
}

func distributorRequestShouldStop(elev config.ElevatorDistributor) bool {
	switch {
	case elev.Direction == config.Down:
		return elev.Requests[elev.Floor][int(elevio.BT_HallDown)] == config.Confirmed ||
			elev.Requests[elev.Floor][int(elevio.BT_Cab)] == config.Confirmed ||
			!distributorRequestsBelow(elev)
	case elev.Direction == config.Up:
		return elev.Requests[elev.Floor][int(elevio.BT_HallUp)] == config.Confirmed ||
			elev.Requests[elev.Floor][int(elevio.BT_Cab)] == config.Confirmed ||
			!distributorRequestsAbove(elev)
	default:
		return true
	}
}

func distributorRequestChooseDirection(elev *config.ElevatorDistributor) {
	switch elev.Direction {
	case config.Up:
		if distributorRequestsAbove(*elev) {
			elev.Direction = config.Up
		} else if distributorRequestsBelow(*elev) {
			elev.Direction = config.Down
		} else {
			elev.Direction = config.Stop
		}
	case config.Down:
		fallthrough
	case config.Stop:
		if distributorRequestsBelow(*elev) {
			elev.Direction = config.Down
		} else if distributorRequestsAbove(*elev) {
			elev.Direction = config.Up
		} else {
			elev.Direction = config.Stop
		}
	}
}
