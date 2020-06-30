package predicate

import "sigs.k8s.io/controller-runtime/pkg/event"

type HasLabels struct {
	Labels map[string]string
}

func (p HasLabels) check(labels map[string]string) bool {
	for k, v := range p.Labels {
		_, ok := labels[k]
		if !ok || labels[k] != v {
			return false
		}
	}
	return true
}

func (p HasLabels) Create(event event.CreateEvent) bool {
	return p.check(event.Meta.GetLabels())
}

func (p HasLabels) Delete(event event.DeleteEvent) bool {
	return p.check(event.Meta.GetLabels())
}

func (p HasLabels) Update(event event.UpdateEvent) bool {
	return p.check(event.MetaNew.GetLabels())
}

func (p HasLabels) Generic(event event.GenericEvent) bool {
	return p.check(event.Meta.GetLabels())
}
