package queue

type Service interface {
	Add(interface{}) error
	Consume() interface{}
	SetIsConsuming(bool)
	GetIsConsuming() bool
}
