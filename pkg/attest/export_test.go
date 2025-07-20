package attest

import "strconv"

// Do

func (do *Do) Cancel() {
	do.cancel()
}

func (do *Do) MockProcess(name, realPort string) {
	proc := &Process{}

	proc.realPort, _ = strconv.Atoi(realPort)

	do.processes.Set(name, proc)
}
