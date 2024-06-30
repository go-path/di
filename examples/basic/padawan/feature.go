package padawan

import "github.com/go-path/di"

type JediService interface {
	FeelTheForce()
}

type PadawanController struct {
	Master JediService `inject:""`
}

func (p *PadawanController) Initialize() {
	println("[Padawan] Master, I want to learn the ways of the force...")
	p.Master.FeelTheForce()
}

func init() {
	// register as startup component, injecting dependencies
	di.Register(di.Injected[*PadawanController](), di.Startup(100))
}
