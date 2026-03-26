package bootstrap

type Hook interface {
	hook()
}

type In struct{}

type CloseHook struct {
	Target any
}

func (CloseHook) hook() {}

func Close(target any) Hook {
	return CloseHook{Target: target}
}

type StartStopHook struct {
	Start any
	Stop  any
}

func (StartStopHook) hook() {}

func StartStop(start any, stop any) Hook {
	return StartStopHook{
		Start: start,
		Stop:  stop,
	}
}

type FuncHook struct {
	Start any
	Stop  any
}

func (FuncHook) hook() {}

func HookFunc(start any, stop any) Hook {
	return FuncHook{
		Start: start,
		Stop:  stop,
	}
}
