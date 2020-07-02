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
	}, []string{"policy", "type"})
	reqAuthdCount    = reqCount.MustCurryWith(prometheus.Labels{"type": "authenticated"})
	reqUnauthdCount  = reqCount.MustCurryWith(prometheus.Labels{"type": "unauthenticated"})
	reqCallbackCount = reqCount.MustCurryWith(prometheus.Labels{"type": "callback"})
	reqExpiredCount  = reqCount.MustCurryWith(prometheus.Labels{"type": "expired"})

	resCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "auth",
		Name:      "responses",
		Help:      "Total number of authorization responses",
	}, []string{"policy", "result"})
	resAllowedCount = resCount.MustCurryWith(prometheus.Labels{"result": "allowed"})
	resDeniedCount  = resCount.MustCurryWith(prometheus.Labels{"result": "denied"})
	resRedirCount   = resCount.MustCurryWith(prometheus.Labels{"result": "redirected"})
	resBadReqCount  = resCount.MustCurryWith(prometheus.Labels{"result": "bad-request"})
	resErrorCount   = resCount.MustCurryWith(prometheus.Labels{"result": "error"})
	resOtherCount   = resCount.MustCurryWith(prometheus.Labels{"result": "other"})
)
