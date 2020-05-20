package auth

import (
	"errors"
	"google.golang.org/protobuf/types/known/structpb"
)

// TODO: Add always deny option
type authorization struct {
	service string
	roles   map[string][]string
}

func marshallAuthz(authz *authorization) *structpb.Struct {
	meta := mkStructValue(1).GetStructValue()
	meta.GetFields()["config"] = mkStructValue(3)
	cfg := meta.GetFields()["config"].GetStructValue().GetFields()

	cfg["service"] = mkStringValue(authz.service)

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

	cfg := meta.Fields["config"].GetStructValue().GetFields()

	authz := &authorization{}
	authz.service = cfg["service"].GetStringValue()

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

func mkStringValue(str string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: str,
		},
	}
}
