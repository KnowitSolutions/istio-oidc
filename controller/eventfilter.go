package controller

import (
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type eventFilter struct {
	predicate.Funcs
	trackedType reflect.Type
	tracked     map[types.UID]bool
	mark        bool
}

func newEventFilter(example runtime.Object) *eventFilter {
	filt := eventFilter{
		trackedType: reflect.TypeOf(example),
		tracked: map[types.UID]bool{},
	}
	filt.CreateFunc = filt.create
	filt.UpdateFunc = filt.update
	filt.DeleteFunc = filt.delete
	return &filt
}

func (ef *eventFilter) track(meta meta.Object) {
	ef.tracked[meta.GetUID()] = ef.mark
}

func (ef *eventFilter) clean() {
	for k, v := range ef.tracked {
		if v != ef.mark {
			delete(ef.tracked, k)
		}
	}

	ef.mark = !ef.mark
}

func (ef *eventFilter) filter(meta meta.Object, obj runtime.Object) bool {
	if reflect.TypeOf(obj) != ef.trackedType {
		return true
	}

	uid := meta.GetUID()
	for elem := range ef.tracked {
		if elem == uid {
			return true
		}
	}

	return false
}

func (ef *eventFilter) create(event event.CreateEvent) bool {
	return ef.filter(event.Meta, event.Object)
}

func (ef *eventFilter) delete(event event.DeleteEvent) bool {
	delete(ef.tracked, event.Meta.GetUID())
	return ef.filter(event.Meta, event.Object)
}

func (ef *eventFilter) update(event event.UpdateEvent) bool {
	return ef.filter(event.MetaOld, event.ObjectOld)
}
