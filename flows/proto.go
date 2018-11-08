package flows

type Proto interface {
	Run()
	Shutdown()
}
