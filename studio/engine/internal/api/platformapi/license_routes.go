package platformapi

import "net/http"

// licenseStatus is GET /license/status: the status plus app_version. It is not wrapped in
// _run, so a failure reaches the app's handlers.
func (a *Area) licenseStatus(r *http.Request) (any, error) {
	status, err := a.deps.License.Status()
	if err != nil {
		return nil, a.uncaught(err)
	}
	return status.Set("app_version", a.deps.AppVersion), nil
}

// licenseTrial is POST /license/trial (no body is read).
func (a *Area) licenseTrial(r *http.Request) (any, error) {
	return a.licenseRun(a.deps.License.StartTrial())
}

// licenseActivate is POST /license/activate: {"license_key", "move"}; the body is read
// before the action runs, as `body = _body()` comes first in Python.
func (a *Area) licenseActivate(r *http.Request) (any, error) {
	body, err := dictBody(r)
	if err != nil {
		return nil, err
	}
	key, _ := get(body, "license_key", "")
	move, _ := get(body, "move", nil)
	return a.licenseRun(a.deps.License.Activate(key, truthy(move)))
}

// licenseActivateOffline is POST /license/activate-offline: {"certificate"}; Python reads
// the body inside _run, where a 413 is not a license error either.
func (a *Area) licenseActivateOffline(r *http.Request) (any, error) {
	body, err := dictBody(r)
	if err != nil {
		return nil, err
	}
	certificate, _ := get(body, "certificate", "")
	return a.licenseRun(a.deps.License.ActivateOffline(certificate))
}

// licenseConfig is PUT /license/config: {"server_url"}.
func (a *Area) licenseConfig(r *http.Request) (any, error) {
	body, err := dictBody(r)
	if err != nil {
		return nil, err
	}
	url, _ := get(body, "server_url", "")
	return a.licenseRun(a.deps.License.SetServerURL(url))
}
