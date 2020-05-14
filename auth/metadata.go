package auth

import (
	"errors"
	"github.com/apex/log"
	"google.golang.org/protobuf/types/known/structpb"
)

// TODO: Get rid of all this in favour of proto values
type metadata map[string]metadataNamespace
type metadataNamespace map[string]metadataValue
type metadataValue struct {
	float64
	string
}

func mapMetadata(m map[string]*structpb.Struct) (metadata, error) {
	if m == nil {
		return metadata{}, nil
	}

	data := make(metadata, len(m))
	for ns, nsV := range m {
		data[ns] = make(metadataNamespace, len(nsV.Fields))
		for k, v := range nsV.Fields {
			switch wrap := v.Kind.(type) {
			case *structpb.Value_NumberValue:
				data[ns][k] = metadataValue{float64: wrap.NumberValue}
			case *structpb.Value_StringValue:
				data[ns][k] = metadataValue{string: wrap.StringValue}
			default:
				log.WithFields(log.Fields{
					"namespace": ns,
					"key": k,
					"value": v,
				}).Error("Invalid metadata type")
				return nil, errors.New("invalid type")
			}
		}
	}

	return data, nil
}
