package main

import (
	"encoding/base64"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	_struct "github.com/golang/protobuf/ptypes/struct"
)

func main() {
	str := `{
		"service": "test",
		"roles": {
			"": ["test1"],
			"test": ["test2"]
		}
	}`

	val := &_struct.Value{}
	_ = jsonpb.UnmarshalString(str, val)

	bin, _ := proto.Marshal(val)

	b64 := make([]byte, base64.StdEncoding.EncodedLen(len(bin)))
	base64.StdEncoding.Encode(b64, bin)

	print(string(b64))
}
