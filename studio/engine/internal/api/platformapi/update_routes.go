package platformapi

import "net/http"

// updateStatus is GET /app/update: the stored answer, read again.
func (a *Area) updateStatus(r *http.Request) (any, error) {
	status, err := a.deps.Updates.Status()
	if err != nil {
		return nil, uncaughtKeyError(err)
	}
	return status, nil
}

// updateCheck is POST /app/update/check: always a manual check (the body is ignored).
func (a *Area) updateCheck(r *http.Request) (any, error) {
	status, err := a.deps.Updates.Check(true)
	if err != nil {
		return nil, uncaughtKeyError(err)
	}
	return status, nil
}

// updateConfigure is PUT /app/update: the raw body (None when not JSON) goes to configure;
// its ValueError answers 400 {"error", "code": "invalid"}.
func (a *Area) updateConfigure(r *http.Request) (any, error) {
	changes, err := silentJSON(r)
	if err != nil {
		return nil, err
	}
	status, err := a.deps.Updates.Configure(changes)
	if err != nil {
		return nil, configFailure(err)
	}
	return status, nil
}
