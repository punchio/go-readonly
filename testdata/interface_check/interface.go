package interface_check

type iGet interface {
	Get() []int
}

type iRoGet interface {
	Get() (ro []int)
}
