package auth

import (
	"context"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/metadata"
	"net/http"
)

func tracingCtx(ctx context.Context) context.Context {
	client := &http.Client{Transport: &TracingMiddleware{Next: http.DefaultTransport}}
	return context.WithValue(ctx, oauth2.HTTPClient, client)
}

type TracingMiddleware struct {
	Next http.RoundTripper
}

var fwd = []string{
	"x-request-id",
	"x-b3-traceid",
	"x-b3-spanid",
	"x-b3-parentspanid",
	"x-b3-sampled",
	"x-b3-flags",
	"x-ot-span-context",
}

func (t *TracingMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {
	meta, _ := metadata.FromIncomingContext(req.Context())
	if meta != nil {
	}

	for _, k := range fwd {
		for _, v := range meta[k] {
			req.Header.Add(k, v)
		}
	}

	return t.Next.RoundTrip(req)
}
