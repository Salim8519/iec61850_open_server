package selftest

import (
	"net/http"
	"os"
	"time"

	"studio-engine/internal/app"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// recoveryWait is recovered.extensions["startup"].wait(120).
const recoveryWait = 120 * time.Second

// recoveryRun is the second half of check(): a new engine on the same folder must restore
// the saved project stopped, then delete the IED (with its test-panel entry) and restore
// it from the recovery file the deletion wrote. The engine is closed afterwards.
func (r *run) recoveryRun() error {
	recovered, err := r.compose("isolated-recovery-test")
	if err != nil {
		return err
	}
	err = r.verifyRecovery(recovered)
	if closed := recovered.Close(); err == nil {
		err = closed
	}
	return err
}

func (r *run) verifyRecovery(recovered *app.Engine) error {
	if !recovered.Recovery.Wait(recoveryWait) {
		return failed("startup recovery did not finish within 120 seconds", "")
	}
	if !pyval.Equal(pyval.Get(recovered.Projects.Project(), "id"), r.identity) {
		return failed("startup recovery opened another project", "")
	}
	result, err := recovered.Recovery.Result()
	if err != nil {
		return err
	}
	if pyval.Str(result, "status") != "restored" {
		return failed("startup recovery did not restore the saved project", "")
	}
	if recovered.Runtime.State() != "stopped" {
		return failed("the recovered simulation is not stopped", "")
	}
	if len(recovered.Runtime.Running()) > 0 {
		return failed("the recovered engine started native workers", "")
	}
	client := newClient(recovered.Handler)
	session, err := client.sendJSON(http.MethodPost, "/api/v1/session", pyval.Obj("token", "isolated-recovery-test"))
	if err != nil {
		return err
	}
	if err := expectStatus(session, http.StatusOK, "POST /api/v1/session did not answer 200", false); err != nil {
		return err
	}
	project := recovered.Projects.Project()
	ied := firstIED(project)
	if ied == nil {
		return failed("the recovered project has no IED", "")
	}
	key := pyval.Str(ied, "id")
	if !panelRefers(project, alarm) {
		return failed("the recovered test panel lost "+alarm, "")
	}
	preview := client.get("/api/v1/ieds/" + key + "/removal-preview")
	if err := expectStatus(preview, http.StatusOK, "the IED removal preview did not answer 200", false); err != nil {
		return err
	}
	return r.deleteAndRestore(recovered, client, key)
}

// panelRefers is project["test_panel"][0]["ref"] == ref.
func panelRefers(project *pyjson.Object, ref string) bool {
	panel := pyval.List(project, "test_panel")
	if len(panel) == 0 {
		return false
	}
	entry, _ := panel[0].(*pyjson.Object)
	return pyval.Str(entry, "ref") == ref
}

// deleteAndRestore deletes the IED (its test-panel entry goes with it) and restores the
// project from the recovery file the deletion wrote inside the temporary folder.
func (r *run) deleteAndRestore(recovered *app.Engine, client *client, key string) error {
	deleted, err := client.sendJSON(http.MethodDelete, "/api/v1/ieds/"+key, pyval.Obj("confirm", true))
	if err != nil {
		return err
	}
	if err := expectStatus(deleted, http.StatusOK, "DELETE /api/v1/ieds/<id> did not answer 200", true); err != nil {
		return err
	}
	project := recovered.Projects.Project()
	if !emptyList(pyval.Get(project, "ieds")) {
		return failed("the deleted IED is still in the project", "")
	}
	if !emptyList(pyval.Get(project, "test_panel")) {
		return failed("the deleted IED's test-panel entry is still in the project", "")
	}
	body, _ := deleted.json().(*pyjson.Object)
	backup := pyval.Str(body, "recovery_file")
	if !within(backup, r.directory) {
		return failed("the recovery file is outside the self-test folder", "")
	}
	payload, err := os.ReadFile(backup)
	if err != nil {
		return err
	}
	if err := recovered.Projects.Restore(payload, true); err != nil {
		return err
	}
	project = recovered.Projects.Project()
	if restored := firstIED(project); restored == nil || pyval.Str(restored, "id") != key {
		return failed("restoring the recovery file did not bring the IED back", "")
	}
	if len(pyval.List(project, "test_panel")) != 1 {
		return failed("restoring the recovery file did not bring the test-panel entry back", "")
	}
	return nil
}
