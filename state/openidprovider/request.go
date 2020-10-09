package openidprovider

import (
	"context"
	"encoding/json"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"net/http"
)

func doJsonRequest(ctx context.Context, url string, data interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return errors.Wrap(err, "failed preparing request", "url", url)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "communication error", "url", url)
	}
	defer func() { _ = res.Body.Close() }()

	err = json.NewDecoder(res.Body).Decode(data)
	if err != nil {
		return errors.Wrap(err, "failed decoding JSON", "url", url)
	}

	return nil
}
