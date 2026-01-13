package events

// placeholder for event data
type Event any

// sample events
type DeviceAck struct{
	AckCode string
}
type DeviceReject struct{}
type Timeout struct{}

type StartConfig struct{}

type DeviceCommand interface {}
type StartConfigCommand struct{}
type EndConfigCommand struct{}
type SetSleepPeriodCommand struct{}
type EndConfigAck struct{}
type ValueUnlockCommand struct{}
type SetProtectedValueCommand struct{}
type ValueLockCommand struct{}