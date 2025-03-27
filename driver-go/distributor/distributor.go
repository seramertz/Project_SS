package distributor

import (
	"Driver-go/assigner"
	"Driver-go/config"
	"Driver-go/local_elevator/elevator"
	"Driver-go/elevio"
	"Driver-go/network/peers"
	"time"
	"strconv"
)

// Initialize elevator distributor
func elevatorDistributorInit(id string) config.ElevatorDistributor{
	requests := make([][]config.RequestState, 4)
	for floor := range requests{
		requests[floor] = make([]config.RequestState, 3)
	}
	return config.ElevatorDistributor{Requests: requests, ID: id, Floor:0, Behaviour: config.Idle}

}

// Distribuing orders among the elevators
func Distributor(
	id 						int,
	ch_newLocalOrder 		chan elevio.ButtonEvent,
	ch_newLocalState 		chan elevator.Elevator, 
	ch_msgFromNetwork 		chan []config.ElevatorDistributor, 
	ch_msgToNetwork 		chan []config.ElevatorDistributor, 
	ch_orderToLocal 		chan elevio.ButtonEvent, 
	ch_peerUpdate 			chan peers.PeerUpdate, 
	ch_watchdogStuckReset 	chan bool , 
	ch_watchdogStuckSignal 	chan bool, 
	ch_clearLocalHallOrders chan bool){


	elevators := make([]*config.ElevatorDistributor, 0)
	thisElevator := new(config.ElevatorDistributor)
	*thisElevator = elevatorDistributorInit(strconv.Itoa(id))
	elevators = append(elevators, thisElevator)

	connectTimer := time.NewTimer(time.Duration(config.ReconnectTimer)*time.Second)

	// Check the network for new elevators, handles receiving the new elevators states
	select{
	case newElevators := <- ch_msgFromNetwork:
		for _, elev := range newElevators{
			if elev.ID == elevators[config.LocalElevator].ID{
				for floor := range elev.Requests{
					if (elev.Requests[floor][config.Cab] == config.Confirmed) || 
						(elev.Requests[floor][config.Cab] == config.Order){
						ch_newLocalOrder <- elevio.ButtonEvent{Floor: floor, Button: elevio.ButtonType(int(config.Cab))}
					}
				}
			}
		}
		break 

	case <- connectTimer.C:
		break
	}

	// Distributes orders among the elevators on the network
	for{
		select{
		case newOrder := <- ch_newLocalOrder:
			assigner.AssignOrder(elevators, newOrder)
			if elevators[config.LocalElevator].Requests[newOrder.Floor][newOrder.Button] == config.Order{
				broadcastElevatorState(elevators, ch_msgToNetwork)
				elevators[config.LocalElevator].Requests[newOrder.Floor][newOrder.Button] = config.Confirmed
				setElevatorLights(elevators,id)
				ch_orderToLocal <- newOrder
			}
			broadcastElevatorState(elevators, ch_msgToNetwork)
			setElevatorLights(elevators,id)

		case newState := <- ch_newLocalState:
			if (newState.Floor != elevators[config.LocalElevator].Floor) || 
				(newState.Behave == elevator.Idle) || 
				(newState.Behave == elevator.DoorOpen) {
				elevators[config.LocalElevator].Behaviour = config.Behaviour(int(newState.Behave))
				elevators[config.LocalElevator].Floor = newState.Floor
				elevators[config.LocalElevator].Direction = config.Direction(int(newState.Direction))
				ch_watchdogStuckReset <- false
			}
			for floor := range elevators[config.LocalElevator].Requests{
				for button := range elevators[config.LocalElevator].Requests[floor]{
					if (!newState.Requests[floor][button]) && 
						(elevators[config.LocalElevator].Requests[floor][button] == config.Confirmed){
						elevators[config.LocalElevator].Requests[floor][button] = config.Complete
					}
					if (elevators[config.LocalElevator].Behaviour != config.Unavailable) && 
						(newState.Requests[floor][button]) &&
						 (elevators[config.LocalElevator].Requests[floor][button] != config.Confirmed){
						elevators[config.LocalElevator].Requests[floor][button] = config.Confirmed
					}
				}
				
			}
			setElevatorLights(elevators,id)
			broadcastElevatorState(elevators, ch_msgToNetwork)
			assigner.RemoveCompletedOrders(elevators)
			
		case newElevators := <- ch_msgFromNetwork:
			if len(newElevators) > 0 {
				updateElevators(elevators,newElevators)
			}
			assigner.ReassignOrders(elevators, ch_newLocalOrder)
			for _, newElev := range newElevators{
				elevExists := false
				for _, elev := range elevators{
					if elev.ID == newElev.ID{
						elevExists = true
						break
					}
				}
				if !elevExists{
					addNewElevator(&elevators,newElev)
				}
			}
			extractNewOrder := assigner.ConfirmedNewOrder(elevators[config.LocalElevator])
			setElevatorLights(elevators,id)
			assigner.RemoveCompletedOrders(elevators)
			if extractNewOrder != nil{
				tempOrder := elevio.ButtonEvent{
					Button : elevio.ButtonType(extractNewOrder.Button),
					Floor : extractNewOrder.Floor}
				ch_orderToLocal <- tempOrder
				broadcastElevatorState(elevators, ch_msgToNetwork)
			}

		case peer := <- ch_peerUpdate:
			if len(peer.Lost) != 0{
				for _, stringLostId := range peer.Lost{
					for _,elev := range elevators {
						if stringLostId == elev.ID{
							elev.Behaviour = config.Unavailable
						}
						assigner.ReassignOrders(elevators, ch_newLocalOrder)
						for floor := range elev.Requests{
							for button := 0; button < len(elev.Requests[floor])-1 ; button++{
								elev.Requests[floor][button] = config.None
							}
						}
					}
				}
			}
			setElevatorLights(elevators,id)
			broadcastElevatorState(elevators, ch_msgToNetwork)

		case <- ch_watchdogStuckSignal: // Detection of stuck elevator
			elevators[config.LocalElevator].Behaviour = config.Unavailable
			broadcastElevatorState(elevators, ch_msgToNetwork)
			for floor := range elevators[config.LocalElevator].Requests{
				for button := 0; button < len(elevators[config.LocalElevator].Requests[floor])-1;button++{
					elevators[config.LocalElevator].Requests[floor][button] = config.None
				}
			}
			setElevatorLights(elevators,id)
			ch_clearLocalHallOrders <- true
			reinitializeElevator(elevators, id)
            broadcastElevatorState(elevators, ch_msgToNetwork)

		}
	}
}

