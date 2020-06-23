package auth

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	reqCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "auth",
		Name:      "requests",
		Help:      "Total number of authorization requests",
	}, []string{"type"})
	reqAuthdCount    = reqCount.WithLabelValues("authenticated")
	reqUnauthdCount  = reqCount.WithLabelValues("unauthenticated")
	reqCallbackCount = reqCount.WithLabelValues("callback")
	reqExpiredCount  = reqCount.WithLabelValues("expired")
	reqBadReqCount   = reqCount.WithLabelValues("bad-request")

	resCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "auth",
		Name:      "responses",
		Help:      "Total number of authorization responses",
	}, []string{"result"})
	resAllowed = resCount.WithLabelValues("allowed")
	resDenied  = resCount.WithLabelValues("denied")
	resRedir   = resCount.WithLabelValues("redirected")
	resBadReq  = resCount.WithLabelValues("bad-request")
	resError   = resCount.WithLabelValues("error")
	resOther   = resCount.WithLabelValues("other")
)
