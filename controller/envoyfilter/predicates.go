package envoyfilter

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"strings"
)

type inNamespace struct {
	Namespace string
}

func (p inNamespace) Create(event event.CreateEvent) bool {
	return event.Meta.GetNamespace() == p.Namespace
}

func (p inNamespace) Delete(event event.DeleteEvent) bool {
	return event.Meta.GetNamespace() == p.Namespace
}

func (p inNamespace) Update(event event.UpdateEvent) bool {
	return event.MetaNew.GetNamespace() == p.Namespace
}

func (p inNamespace) Generic(event event.GenericEvent) bool {
	return event.Meta.GetNamespace() == p.Namespace
}

type hasPrefix struct {
	Prefix string
}

func (p hasPrefix) Create(event event.CreateEvent) bool {
	return strings.HasPrefix(event.Meta.GetName(), p.Prefix)
}

func (p hasPrefix) Delete(event event.DeleteEvent) bool {
	return strings.HasPrefix(event.Meta.GetName(), p.Prefix)
}

func (p hasPrefix) Update(event event.UpdateEvent) bool {
	return strings.HasPrefix(event.MetaNew.GetName(), p.Prefix)
}

func (p hasPrefix) Generic(event event.GenericEvent) bool {
	return strings.HasPrefix(event.Meta.GetName(), p.Prefix)
}
