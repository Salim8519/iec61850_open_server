package updates

import (
	"io"
	"net/http"
	"strings"
)

// fetch is UpdateService._fetch: GET <server>/api/v1/release with If-None-Match. It sends
// nothing about this PC (User-Agent "ScadaBench" and Accept only). raw nil means 304 Not
// Modified (the stored etag stays). A *fetchError carries the stored error code; other
// errors are ones Python does not catch (the license config could not be read).
func (s *Service) fetch(etag any) (raw []byte, newTag any, err error) {
	url, err := s.options.ServerURL()
	if err != nil {
		return nil, nil, err
	}
	url = strings.TrimRight(url, "/")
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://127.0.0.1") &&
		!strings.HasPrefix(url, "http://localhost") {
		return nil, nil, &fetchError{"no_server"}
	}
	request, err := http.NewRequest("GET", url+"/api/v1/release", nil)
	if err != nil {
		return nil, nil, &fetchError{"offline"}
	}
	request.Header.Set("User-Agent", "ScadaBench")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Encoding", "identity")
	if text, isText := etag.(string); isText && text != "" {
		request.Header.Set("If-None-Match", text)
	}
	response, err := s.options.Opener.Do(request)
	if err != nil {
		return nil, nil, &fetchError{"offline"}
	}
	defer response.Body.Close()
	switch {
	case response.StatusCode == http.StatusNotModified:
		return nil, etag, nil
	case response.StatusCode == http.StatusNotFound:
		return nil, nil, &fetchError{"no_release"}
	case response.StatusCode < 200 || response.StatusCode > 299:
		return nil, nil, &fetchError{"server_error"}
	}
	raw, err = io.ReadAll(io.LimitReader(response.Body, MaxBody+1))
	if err != nil {
		return nil, nil, &fetchError{"offline"}
	}
	if values := response.Header.Values("Etag"); len(values) > 0 {
		newTag = values[0]
	}
	return raw, newTag, nil
}
