package auth

import (
	"errors"
	"google.golang.org/protobuf/types/known/structpb"
)

type authorization struct {
	service string
	blocked bool
	roles   map[string][]string
}

func marshallAuthz(authz *authorization) *structpb.Struct {
	meta := mkStructValue(1).GetStructValue()
	meta.GetFields()["config"] = mkStructValue(3)
	cfg := meta.GetFields()["config"].GetStructValue().GetFields()

	cfg["service"] = mkStringValue(authz.service)
	cfg["blocked"] = mkBoolValue(authz.blocked)

	cfg["roles"] = mkStructValue(len(authz.roles))
	dict := cfg["roles"].GetStructValue().GetFields()
	for k, v := range authz.roles {
		dict[k] = mkListValue(len(v))
		vals := dict[k].GetListValue().GetValues()
		for i, v := range v {
			vals[i] = mkStringValue(v)
		}
	}

	return meta
}

func unmarshallAuthz(meta *structpb.Struct) (*authorization, error) {
	if meta == nil {
		return nil, errors.New("missing config")
	}

	authz := &authorization{}
	cfg := meta.Fields["config"].GetStructValue().GetFields()

	authz.service = cfg["service"].GetStringValue()
	authz.blocked = cfg["blocked"].GetBoolValue()

	dict := cfg["roles"].GetStructValue().GetFields()
	authz.roles = make(map[string][]string, len(dict))
	for k, v := range dict {
		vals := v.GetListValue().GetValues()
		authz.roles[k] = make([]string, len(vals))
		for i, v := range vals {
			authz.roles[k][i] = v.GetStringValue()
		}
	}

	return authz, nil
}

func mkStructValue(size int) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{
			StructValue: &structpb.Struct{
				Fields: make(map[string]*structpb.Value, size),
			},
		},
	}
}

func mkListValue(size int) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_ListValue{
			ListValue: &structpb.ListValue{
				Values: make([]*structpb.Value, size),
			},
		},
	}
}

func mkBoolValue(value bool) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_BoolValue{
			BoolValue: value,
		},
	}
}


func mkStringValue(value string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: value,
		},
	}
}
