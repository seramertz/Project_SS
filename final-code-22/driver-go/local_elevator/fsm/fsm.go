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
	ch_orderChan 			chan elevio.ButtonEvent,
	ch_elevatorState 		chan<- elevator.Elevator,
	ch_clearLocalHallOrders chan bool,
	ch_arrivedAtFloors 		chan int,
	ch_obstruction 			chan bool,
	ch_timerDoor 			chan bool) {

	// Initialize elevator
	e := elevator.InitElevator()
	elev := &e
	elevio.SetDoorOpenLamp(false)
	ch_elevatorState <- *elev

	doorTimer := time.NewTimer(time.Duration(config.DoorOpenDuration) * time.Second)
	timerUpdateState := time.NewTicker(time.Duration(config.StateUpdatePeriodsMs) * time.Millisecond)

	// Finite State Machine: Handles elevator behavior based on different events
	for {
		elevator.SetLocalLights(*elev)
		select {
		case order := <-ch_orderChan: // Handles new orders
			switch {
			case elev.Behaviour == elevator.DoorOpen:
				if elev.Floor == order.Floor {
					// If the door is open and the order is for the current floor, reset door timer
					doorTimer.Reset(time.Duration(config.DoorOpenDuration) * time.Second)
				} else {
					// Otherwise, register the request
					elev.Requests[order.Floor][order.Button] = true
				}

			case elev.Behaviour == elevator.Moving:
				// If the elevator is moving, store the request to be handled later
				elev.Requests[order.Floor][order.Button] = true

			case elev.Behaviour == elevator.Idle:
				if elev.Floor == order.Floor {
					// If the elevator is idle and already at the requested floor, open the door
					elevio.SetDoorOpenLamp(true)
					doorTimer.Reset(time.Duration(config.DoorOpenDuration) * time.Second)
					elev.Behaviour = elevator.DoorOpen
					ch_elevatorState <- *elev
				} else {
					// Otherwise, store the request and start moving
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

		case <-doorTimer.C: // Handles door closing logic
			switch {
			case elev.Behaviour == elevator.DoorOpen:
				if elev.Obstructed {
					elevio.SetMotorDirection(elevio.MD_Stop)
					doorTimer.Stop()
				} else {
					request.RequestChooseDirection(elev)
					elevio.SetMotorDirection(elev.Direction)
					elevio.SetDoorOpenLamp(false)

					// Transition to the next state
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

		case <-ch_clearLocalHallOrders: // Clear hallorders of this elevator
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

		case <-timerUpdateState.C: // Periodic state update
			ch_elevatorState <- *elev
			timerUpdateState.Reset(time.Duration(config.StateUpdatePeriodsMs) * time.Millisecond)

		}
	}
}
