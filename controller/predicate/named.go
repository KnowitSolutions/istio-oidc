package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type IsNamed struct {
	Name string
}

func (p IsNamed) Create(event event.CreateEvent) bool {
	return event.Meta.GetName() == p.Name
}

func (p IsNamed) Delete(event event.DeleteEvent) bool {
	return event.Meta.GetName()== p.Name
}

func (p IsNamed) Update(event event.UpdateEvent) bool {
	return event.MetaNew.GetName() == p.Name
}

func (p IsNamed) Generic(event event.GenericEvent) bool {
	return event.Meta.GetName() == p.Name
}
