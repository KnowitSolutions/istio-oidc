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
	*server
}

func (srv *ServerV2) Check(ctx context.Context, req *auth.CheckRequest) (*auth.CheckResponse, error) {
	proto := req.Attributes.Request.Http.Headers["x-forwarded-proto"]
	host := req.Attributes.Request.Http.Host
	path := req.Attributes.Request.Http.Path
	addr := fmt.Sprintf("%s://%s%s", proto, host, path)
	cookies := req.Attributes.Request.Http.Headers["cookie"]
	meta := req.Attributes.ContextExtensions
	data, err := srv.newRequest(addr, cookies, meta)

	var r *response
	if err == nil {
		r = srv.check(ctx, data)
	} else {
		log.WithError(err).Error("Unable to construct request object")
		r = &response{status: http.StatusInternalServerError}
	}

	res := &auth.CheckResponse{Status: &status.Status{}}

	// TODO: Clean up which codes are returned
	switch r.status {
	case http.StatusOK:
		res.Status.Code = int32(code.Code_OK)
	case http.StatusBadRequest:
		res.Status.Code = int32(code.Code_INVALID_ARGUMENT)
	case http.StatusUnauthorized:
		res.Status.Code = int32(code.Code_UNAUTHENTICATED)
	case http.StatusForbidden:
		res.Status.Code = int32(code.Code_PERMISSION_DENIED)
	case http.StatusInternalServerError:
		res.Status.Code = int32(code.Code_INTERNAL)
	default:
		res.Status.Code = int32(code.Code_UNKNOWN)
	}

	hs := make([]*core.HeaderValueOption, len(r.headers))
	i := 0
	for k, v := range r.headers {
		hs[i] = &core.HeaderValueOption{
			Header: &core.HeaderValue{Key: k, Value: v},
			Append: &wrappers.BoolValue{Value: false},
		}
		i++
	}

	if r.status == http.StatusOK {
		res.HttpResponse = &auth.CheckResponse_OkResponse{
			OkResponse: &auth.OkHttpResponse{
				Headers: hs,
			},
		}
	} else {
		res.HttpResponse = &auth.CheckResponse_DeniedResponse{
			DeniedResponse: &auth.DeniedHttpResponse{
				Status:  &types.HttpStatus{Code: types.StatusCode(r.status)},
				Headers: hs,
			},
		}
	}

	return res, nil
}
