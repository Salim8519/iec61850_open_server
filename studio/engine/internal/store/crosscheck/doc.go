// Package crosscheck holds only tests: the store formats against goldens written by CPython
// (../testdata/fixtures, made by ../testdata/crosscheck.py) and, when STUDIO_PYTHON names
// the studio venv interpreter, a live run of the script that decodes Go-written archives
// with studio_app.projects.archive and regenerates the goldens.
package crosscheck
