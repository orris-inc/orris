package notification

type Specification interface {
	IsSatisfiedBy(entity interface{}) bool
}
