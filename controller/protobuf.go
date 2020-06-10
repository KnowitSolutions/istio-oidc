package controller

import ptypes "github.com/gogo/protobuf/types"

func newStruct(data map[string]interface{}) *ptypes.Struct {
	return newValue(data).Kind.(*ptypes.Value_StructValue).StructValue
}

func newValue(data interface{}) *ptypes.Value {
	ret := &ptypes.Value{}

	switch val := data.(type) {
	case bool:
		ret.Kind = &ptypes.Value_BoolValue{BoolValue: val}

	case string:
		ret.Kind = &ptypes.Value_StringValue{StringValue: val}

	case map[string]interface{}:
		fields := make(map[string]*ptypes.Value, len(val))
		for k, v := range val {
			fields[k] = newValue(v)
		}
		ret.Kind = &ptypes.Value_StructValue{StructValue: &ptypes.Struct{Fields: fields}}

	default:
		panic("unknown datatype")
	}

	return ret
}
