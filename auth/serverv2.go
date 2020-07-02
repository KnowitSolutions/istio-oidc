package auth

import (
	"context"
	"fmt"
	"github.com/apex/log"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	types "github.com/envoyproxy/go-control-plane/envoy/type"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"net/http"
)

type ServerV2 struct {
	*Server
}

func (srv *ServerV2) Check(ctx context.Context, req *auth.CheckRequest) (*auth.CheckResponse, error) {
	ctx = tracingCtx(ctx)

	proto := req.Attributes.Request.Http.Headers["x-forwarded-proto"]
	host := req.Attributes.Request.Http.Host
	path := req.Attributes.Request.Http.Path
	addr := fmt.Sprintf("%s://%s%s", proto, host, path)
	cookies := req.Attributes.Request.Http.Headers["cookie"]
	meta := req.Attributes.ContextExtensions
	data, err := srv.newRequest(addr, cookies, meta)

	var r *response
	if err != nil {
		log.WithError(err).Error("Unable to construct request object")
		r = &response{status: http.StatusBadRequest}
	} else {
		r = srv.check(ctx, data)
	}

	res := &auth.CheckResponse{}
	hs := make([]*core.HeaderValueOption, len(r.headers))

	if r.status == http.StatusOK {
		res.Status = &status.Status{Code: int32(code.Code_OK)}
		res.HttpResponse = &auth.CheckResponse_OkResponse{
			OkResponse: &auth.OkHttpResponse{Headers: hs},
		}
	} else {
		res.Status = &status.Status{Code: int32(code.Code_PERMISSION_DENIED)}
		res.HttpResponse = &auth.CheckResponse_DeniedResponse{
			DeniedResponse: &auth.DeniedHttpResponse{
				Status:  &types.HttpStatus{Code: types.StatusCode(r.status)},
				Headers: hs,
			},
		}
	}

	i := 0
	for k, v := range r.headers {
		hs[i] = &core.HeaderValueOption{
			Header: &core.HeaderValue{Key: k, Value: v},
			Append: &wrappers.BoolValue{Value: false},
		}
		i++
	}

	return res, nil
}
