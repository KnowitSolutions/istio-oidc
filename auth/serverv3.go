package auth

import (
	"context"
	"fmt"
	"github.com/apex/log"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	types "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"net/http"
	"net/url"
)

type ServerV3 struct {
	*server
}

func (srv *ServerV3) Check(_ context.Context, req *auth.CheckRequest) (*auth.CheckResponse, error) {
	r := &response{status: http.StatusInternalServerError}
	fail := false

	meta := req.Attributes.MetadataContext.FilterMetadata["istio-keycloak"]
	authz, err := unmarshallAuthz(meta)
	if err != nil {
		fail = true
		log.WithError(err).Warn("Unable to unmarshall authorization requirements")
	}

	proto := req.Attributes.Request.Http.Headers["x-forwarded-proto"]
	host := req.Attributes.Request.Http.Host
	path := req.Attributes.Request.Http.Path
	loc, err := url.Parse(fmt.Sprintf("%s://%s%s", proto, host, path))
	if err != nil {
		fail = true
		log.WithError(err).Error("Unable to parse request URL")
	}

	dummy := http.Request{Header: http.Header{}}
	dummy.Header.Add("Cookie", req.Attributes.Request.Http.Headers["cookie"])

	if !fail {
		r = srv.check(&request{
			service: srv.services[authz.service],
			url:     *loc,
			cookies: dummy.Cookies(),
			roles:   authz.roles,
		})
	}

	res := &auth.CheckResponse{Status: &status.Status{}}

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
