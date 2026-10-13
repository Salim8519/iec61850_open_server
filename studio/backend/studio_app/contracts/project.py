"""Validate browser-editable project geometry without accepting runtime internals."""
from typing import Annotated, Literal
from pydantic import BaseModel, ConfigDict, Field


class Position(BaseModel):
    x: float = Field(ge=-100000, le=100000, allow_inf_nan=False)
    y: float = Field(ge=-100000, le=100000, allow_inf_nan=False)


class SignalBinding(BaseModel):
    model_config = ConfigDict(extra="forbid")
    ied_id: str
    ref: str
    report_ref: str | None = None


class DirectStateMapping(BaseModel):
    model_config = ConfigDict(extra="forbid")
    mode: Literal["direct"]
    source: SignalBinding


class SinglePointMapping(BaseModel):
    model_config = ConfigDict(extra="forbid")
    mode: Literal["single_to_dbpos"]
    source: SignalBinding


class ContactStateMapping(BaseModel):
    model_config = ConfigDict(extra="forbid")
    mode: Literal["contacts_to_dbpos"]
    open: SignalBinding
    closed: SignalBinding


StateMapping = Annotated[
    DirectStateMapping | SinglePointMapping | ContactStateMapping,
    Field(discriminator="mode"),
]


class ControlBinding(BaseModel):
    model_config = ConfigDict(extra="forbid")
    ied_id: str
    ref: str
    compatibility: Literal["standard", "concen_inverted_sbow"] = "standard"


class RelayStatus(BaseModel):
    model_config = ConfigDict(extra="forbid")
    id: str = Field(pattern=r"^[A-Za-z0-9_-]{1,64}$")
    label: str = Field(min_length=1, max_length=100)
    source: SignalBinding


HEX_COLOR = r"^#[0-9a-fA-F]{6}$"


class EquipmentData(BaseModel):
    model_config = ConfigDict(extra="forbid")
    label: str = Field(min_length=1, max_length=100)
    # TRANSFORMER/LOAD/BUSBAR were retired in format 12 (ADR-0033).
    kind: Literal["CB", "DS", "ES", "CT", "VT", "SOURCE", "RELAY", "TRIP_RELAY", "PUSH_BUTTON", "JUNCTION"]
    rotation: int = 0
    state_mapping: StateMapping | None = None
    control_binding: ControlBinding | None = None
    relay_statuses: list[RelayStatus] = Field(default_factory=list, max_length=50)
    trip_command_polarity: Literal["operate_false", "operate_true"] = "operate_false"
    trip_state_polarity: Literal["operated_false", "operated_true"] = "operated_true"
    push_mode: Literal["spc_momentary", "latching_status", "self_resetting"] = "spc_momentary"
    push_value: bool = True
    push_rest_value: bool = False
    push_reset_ms: int = Field(default=1000, ge=100, le=60000)
    initial_position: Literal["open", "closed"] = "open"
    # Display-only state colours; None means the kind default. Never read by simulation.
    active_color: str | None = Field(default=None, pattern=HEX_COLOR)
    inactive_color: str | None = Field(default=None, pattern=HEX_COLOR)


class EquipmentNode(BaseModel):
    model_config = ConfigDict(extra="ignore")
    id: str = Field(pattern=r"^[A-Za-z0-9_-]{1,64}$")
    type: Literal["equipment"] = "equipment"
    position: Position
    data: EquipmentData


class Wire(BaseModel):
    model_config = ConfigDict(extra="ignore")
    id: str = Field(max_length=128)
    source: str
    target: str
    sourceHandle: str | None = None
    targetHandle: str | None = None


class Diagram(BaseModel):
    nodes: list[EquipmentNode] = Field(default_factory=list, max_length=2000)
    edges: list[Wire] = Field(default_factory=list, max_length=5000)
