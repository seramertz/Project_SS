package fsm

import (
	"Driver-go/config"
	"Driver-go/elevio"
	"Driver-go/local_elevator/elevator"
	"Driver-go/local_elevator/request"
	"time"
)

// Statemachine for running local elevator
func Fsm(
	ch_orderChan chan elevio.ButtonEvent,
	ch_elevatorState chan<- elevator.Elevator,
	ch_clearLocalHallOrders chan bool,
	ch_arrivedAtFloors chan int,
	ch_obstruction chan bool,
	ch_timerDoor chan bool) {

	e := elevator.InitElevator()
	elev := &e

	elevio.SetDoorOpenLamp(false)

	ch_elevatorState <- *elev

	doorTimer := time.NewTimer(time.Duration(config.DoorOpenDuration) * time.Second)
	timerUpdateState := time.NewTicker(time.Duration(config.StateUpdatePeriodsMs) * time.Millisecond)

	// Statemachine defining the elevators state
	for {
		elevator.SetLocalLights(*elev)
		select {
		case order := <-ch_orderChan: // Handles new order
			switch {
			case elev.Behaviour == elevator.DoorOpen:
				if elev.Floor == order.Floor {
					doorTimer.Reset(time.Duration(config.DoorOpenDuration) * time.Second)
				} else {
					elev.Requests[order.Floor][order.Button] = true
				}

			case elev.Behaviour == elevator.Moving:
				elev.Requests[order.Floor][order.Button] = true

			case elev.Behaviour == elevator.Idle:
				if elev.Floor == order.Floor {
					elevio.SetDoorOpenLamp(true)
					doorTimer.Reset(time.Duration(config.DoorOpenDuration) * time.Second)
					elev.Behaviour = elevator.DoorOpen
					ch_elevatorState <- *elev
				} else {
					elev.Requests[order.Floor][order.Button] = true
					request.RequestChooseDirection(elev)
					elevio.SetMotorDirection(elev.Direction)
					elev.Behaviour = elevator.Moving
					ch_elevatorState <- *elev
					break
				}
			}

		case floor := <-ch_arrivedAtFloors: // Handles arriving at floor
			elev.Floor = floor
			switch {
			case elev.Behaviour == elevator.Moving:
				if request.RequestShouldStop(elev) {
					elevio.SetMotorDirection(elevio.MD_Stop)
					request.RequestClearAtCurrentFloor(elev)
					elevio.SetDoorOpenLamp(true)
					doorTimer.Reset(time.Duration(config.DoorOpenDuration) * time.Second)
					elev.Behaviour = elevator.DoorOpen
					ch_elevatorState <- *elev
				}
			default:
				break

			}

		case <-doorTimer.C: // Handles door
			switch {
			case elev.Behaviour == elevator.DoorOpen:
				if elev.Obstructed {
					elevio.SetMotorDirection(elevio.MD_Stop)
					doorTimer.Stop()
				} else {
					request.RequestChooseDirection(elev)
					elevio.SetMotorDirection(elev.Direction)
					elevio.SetDoorOpenLamp(false)
					if elev.Direction == elevio.MD_Stop {
						elev.Behaviour = elevator.Idle
						ch_elevatorState <- *elev
					} else {
						elev.Behaviour = elevator.Moving
						ch_elevatorState <- *elev
					}
				}

			default:
				break
			}

		case <-ch_clearLocalHallOrders: // Delete the hallorders of this elevator
			request.RequestClearHall(elev)

		case obstruction := <-ch_obstruction: // Handles obstruction
			if obstruction {
				elev.Obstructed = true
				elevio.SetDoorOpenLamp(true)
				doorTimer.Stop()
			} else {
				elev.Obstructed = false
				doorTimer.Reset(time.Duration(config.DoorOpenDuration) * time.Second)
			}
			ch_elevatorState <- *elev

		case <-timerUpdateState.C: // Handles timeout
			ch_elevatorState <- *elev
			timerUpdateState.Reset(time.Duration(config.StateUpdatePeriodsMs) * time.Millisecond)

		}
	}
}
