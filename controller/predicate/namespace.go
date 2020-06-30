package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type InNamespace struct {
	Namespace string
}

func (p InNamespace) Create(event event.CreateEvent) bool {
	return event.Meta.GetNamespace() == p.Namespace
}

func (p InNamespace) Delete(event event.DeleteEvent) bool {
	return event.Meta.GetNamespace() == p.Namespace
}

func (p InNamespace) Update(event event.UpdateEvent) bool {
	return event.MetaNew.GetNamespace() == p.Namespace
}

func (p InNamespace) Generic(event event.GenericEvent) bool {
	return event.Meta.GetNamespace() == p.Namespace
}
