package elevator

import (
	"Driver-go/config"
	"Driver-go/elevio"
)

type Behaviour int

const (
	Idle     Behaviour = 0
	DoorOpen Behaviour = 1
	Moving   Behaviour = 2
)

type Elevator struct {
	Floor      int
	Direction  elevio.MotorDirection
	Requests   [][]bool
	Behaviour  Behaviour
	TimerCount int
	Obstructed bool
}

// Initializes an elevator to be at floor zero
func InitElevator() Elevator {
	requests := make([][]bool, 0)
	for floor := 0; floor < config.NumFloors; floor++ {
		requests = append(requests, make([]bool, config.NumButtons))
		for button := range requests[floor] {
			requests[floor][button] = false
		}
	}
	for elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	return Elevator{
		Floor:      elevio.GetFloor(),
		Direction:  elevio.MD_Stop,
		Requests:   requests,
		Behaviour:  Idle,
		TimerCount: 0,
		Obstructed: false}
}

// Set elevtor lights and floor indicators
func SetLocalLights(e Elevator) {
	elevio.SetFloorIndicator(e.Floor)
	for floor := range e.Requests {
		elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), floor, e.Requests[floor][elevio.BT_Cab])
	}
}
