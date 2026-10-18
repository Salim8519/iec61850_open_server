"""Drive simulated subscribers' GOOSE inputs from frames received on the wire (ADR-0066).

The listener (`studio_live`) owns reception and supervision; this class only turns an
accepted frame into ordinary acknowledged samples for the subscriber's worker, so the
native worker stays the owner of its MMS data model and reports.
"""
from studio_app.domain.goose_inputs import accept_frame, bind_inputs, input_updates
from studio_app.domain.goose_supervision import dataset_values
from studio_app.scl.document import parse_document
from studio_app.scl.goose_links import goose_subscriptions


def _with_members(ied, subscription):
    """Projects imported before change 0094 lack dataset positions; read them from the stored SCL."""
    if all("member" in item for item in subscription.get("inputs", [])) and "member_count" in subscription:
        return subscription
    root = parse_document(ied["source"])
    element = next((item for item in root.findall("IED") if item.get("name") == ied["name"]), None)
    found, _ = goose_subscriptions(root, element) if element is not None else ([], [])
    return next((item for item in found if item["gocb_ref"] == subscription["gocb_ref"]), subscription)


class GooseInputs:
    def __init__(self, manager):
        self.manager = manager
        self.routes = {}

    def prepare(self, ieds, watches):
        """Bind inputs once per Run; each watch's subscribers say how many inputs are applied."""
        by_id = {ied["id"]: ied for ied in ieds}
        self.routes = {}
        for watch in watches.values():
            for subscriber in watch["subscribers"]:
                ied = by_id.get(subscriber["ied_id"])
                subscription = next((item for item in (ied or {}).get("goose", {}).get("subscribes", [])
                                     if item["gocb_ref"] == watch["gocb_ref"]), None)
                if subscription is None:
                    continue
                try:
                    subscription = _with_members(ied, subscription)
                except Exception as error:  # A stored source that no longer parses is reported, not fatal.
                    subscriber.update(applied_inputs=0, input_problems=[f"Inputs not applied: {error}"])
                    continue
                bindings, problems = bind_inputs(ied, subscription)
                subscriber.update(applied_inputs=len(bindings), input_problems=problems[:20])
                if bindings:
                    self.routes.setdefault(watch["gocb_ref"], []).append(
                        (ied["id"], bindings, subscription.get("member_count", 0)))

    def apply(self, watch, event):
        routes = self.routes.get(watch["gocb_ref"])
        # A timeout keeps the last applied values, as a subscriber holds its inputs until told otherwise.
        if not routes or event.get("state") == "timeout":
            return
        coordinator = self.manager.coordinator
        publisher_values = self._publisher_values(watch, event)
        for ied_id, bindings, member_count in routes:
            reason = accept_frame(event, member_count)
            if reason:
                coordinator.emit("goose_input_refused", ied_id=ied_id, gocb_ref=watch["gocb_ref"], reason=reason)
                continue
            updates, problems = input_updates(bindings, event["values"], publisher_values)
            applied = []
            if updates and ied_id in coordinator.sample_writer.locks:
                try:
                    coordinator.sample_writer.apply(ied_id, [(ref, value, None) for ref, value in updates],
                                                    "goose_input")
                    applied = [ref for ref, _ in updates]
                except ValueError as error:
                    problems.append(str(error))
            coordinator.emit("goose_input", ied_id=ied_id, gocb_ref=watch["gocb_ref"], st_num=event.get("st_num"),
                             applied=applied, problems=problems[:10])

    def _publisher_values(self, watch, event):
        publisher = watch.get("publisher") or {}
        if not publisher.get("ied_id") or not isinstance(event.get("values"), list):
            return None
        ied = self.manager.coordinator.projects.find_ied(publisher["ied_id"])
        control = next((item for item in ied.get("goose", {}).get("publishes", [])
                        if item["gocb_ref"] == watch["gocb_ref"]), None)
        return dataset_values(ied, control, event["values"]) if control else None
